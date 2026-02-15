package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	twitchIRC "github.com/gempir/go-twitch-irc/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
)

// convertToLiveComment converts a Twitch IRC message to LiveComment format (old chat-downloader format)
func convertToLiveComment(msg twitchIRC.PrivateMessage) utils.LiveComment {
	comment := utils.LiveComment{
		ActionType:       "add_chat_message",
		ChannelID:        msg.RoomID,
		ClientNonce:      msg.ID,
		Colour:           msg.User.Color,
		Flags:            msg.Tags["flags"],
		Message:          msg.Message,
		MessageID:        msg.ID,
		MessageType:      "text",
		ReturningChatter: msg.Tags["returning-chatter"],
		Timestamp:        msg.Time.UnixMicro(),
		UserType:         msg.Tags["user-type"],
	}

	// Parse first-msg tag
	if firstMsg := msg.Tags["first-msg"]; firstMsg == "1" {
		comment.IsFirstMessage = true
	}

	// Set author fields
	comment.Author.DisplayName = msg.User.DisplayName
	comment.Author.ID = msg.User.ID
	comment.Author.Name = msg.User.Name
	comment.Author.IsModerator = msg.User.Badges["moderator"] == 1 || msg.Tags["mod"] == "1"
	comment.Author.IsSubscriber = msg.User.Badges["subscriber"] == 1
	comment.Author.IsTurbo = msg.User.Badges["turbo"] == 1

	// Convert badges
	for badgeName, badgeVersion := range msg.User.Badges {
		badge := struct {
			ClickAction string `json:"click_action"`
			ClickURL    string `json:"click_url"`
			Description string `json:"description"`
			Icons       []struct {
				Height int    `json:"height"`
				ID     string `json:"id"`
				URL    string `json:"url"`
				Width  int    `json:"width"`
			} `json:"icons"`
			ID      string      `json:"id"`
			Name    string      `json:"name"`
			Title   string      `json:"title"`
			Version interface{} `json:"version"`
		}{
			ID:      badgeName,
			Name:    badgeName,
			Title:   badgeName,
			Version: badgeVersion,
		}
		comment.Author.Badges = append(comment.Author.Badges, badge)
	}

	// Convert emotes
	for _, emote := range msg.Emotes {
		commentEmote := struct {
			ID     string `json:"id"`
			Images []struct {
				Height int    `json:"height"`
				ID     string `json:"id"`
				URL    string `json:"url"`
				Width  int    `json:"width"`
			} `json:"images"`
			Locations []string `json:"locations"`
			Name      string   `json:"name"`
		}{
			ID:   emote.ID,
			Name: emote.Name,
		}

		// Convert emote positions to location strings
		for _, position := range emote.Positions {
			location := fmt.Sprintf("%d-%d", position.Start, position.End)
			commentEmote.Locations = append(commentEmote.Locations, location)
		}

		comment.Emotes = append(comment.Emotes, commentEmote)
	}

	return comment
}

// appendMessageToJSONArray appends a message to a JSON array file
// without loading the whole file into memory.
func appendMessageToJSONArray(filename string, comment utils.LiveComment) error {
	// Encode the new comment once.
	msg, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Open (or create) the file for read/write.
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	// Acquire an exclusive advisory lock so concurrent goroutines/processes
	// cannot simultaneously truncate/write the file and corrupt the JSON.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	copyFrom, copyPrefixUntil, isEmptyArray, needsOpeningBracket, err := inspectJSONArrayPrefix(f, info.Size())
	if err != nil {
		return fmt.Errorf("failed to inspect existing JSON array: %w", err)
	}

	// Write to a temp file and atomically rename it over the original.
	// This avoids leaving a partially written/truncated JSON file on crashes.
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := f.Seek(copyFrom, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek original file: %w", err)
	}

	if needsOpeningBracket {
		if _, err := tmp.Write([]byte("[\n")); err != nil {
			return fmt.Errorf("failed to write opening bracket: %w", err)
		}
	}

	if copyPrefixUntil > copyFrom {
		if _, err := io.CopyN(tmp, f, copyPrefixUntil-copyFrom); err != nil {
			return fmt.Errorf("failed to copy existing JSON prefix: %w", err)
		}
	}

	if isEmptyArray {
		if _, err := tmp.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	} else {
		if _, err := tmp.Write([]byte(",\n")); err != nil {
			return fmt.Errorf("failed to write message separator: %w", err)
		}
	}

	if _, err := tmp.Write(msg); err != nil {
		return fmt.Errorf("failed to write chat message: %w", err)
	}
	if _, err := tmp.Write([]byte("\n]\n")); err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to atomically replace file: %w", err)
	}

	// Best-effort sync of parent directory to increase rename durability.
	if dirF, err := os.Open(dir); err == nil {
		_ = dirF.Sync()
		_ = dirF.Close()
	}

	return nil
}

// inspectJSONArrayPrefix inspects the existing file and returns:
//   - copyPrefixUntil: number of bytes from the beginning to copy into the temp file
//     before appending the next message
//   - isEmptyArray: whether the array currently has zero elements
//
// It supports recovery from interrupted writes where trailing commas and/or the
// closing bracket are missing.
func inspectJSONArrayPrefix(f *os.File, size int64) (copyFrom, copyPrefixUntil int64, isEmptyArray, needsOpeningBracket bool, err error) {
	if size == 0 {
		return 0, 0, true, true, nil
	}

	firstIdx, firstByte, ok, err := findFirstNonSpaceInRange(f, 0, size)
	if err != nil {
		return 0, 0, false, false, err
	}
	if !ok {
		// File contains only whitespace; treat as empty/repairable.
		return 0, 0, true, true, nil
	}

	if firstByte != '[' {
		// Recovery path: handle files missing the opening '[' due to prior
		// interrupted/broken writes.
		copyUntil, empty, recErr := inspectMissingOpeningBracketPrefix(f, firstIdx, size)
		if recErr != nil {
			return 0, 0, false, false, recErr
		}
		return firstIdx, copyUntil, empty, true, nil
	}

	lastIdx, lastByte, ok, err := findLastNonSpaceBefore(f, size)
	if err != nil {
		return 0, 0, false, false, err
	}
	if !ok {
		return 0, 0, true, true, nil
	}

	if lastByte == ']' {
		prevIdx, prevByte, ok, err := findLastNonSpaceBefore(f, lastIdx)
		if err != nil {
			return 0, 0, false, false, err
		}
		if !ok {
			return 0, 0, false, false, fmt.Errorf("malformed JSON array")
		}

		return 0, lastIdx, prevIdx == firstIdx && prevByte == '[', false, nil
	}

	// Recovery path: file likely ended mid-write. Trim trailing commas/whitespace.
	searchEnd := size
	for {
		idx, b, found, err := findLastNonSpaceBefore(f, searchEnd)
		if err != nil {
			return 0, 0, false, false, err
		}
		if !found {
			return 0, firstIdx + 1, true, false, nil
		}

		if b == ',' {
			searchEnd = idx
			continue
		}

		copyPrefixUntil = idx + 1
		break
	}

	if copyPrefixUntil <= firstIdx {
		return 0, firstIdx + 1, true, false, nil
	}

	_, _, hasContentAfterOpenBracket, err := findFirstNonSpaceInRange(f, firstIdx+1, copyPrefixUntil)
	if err != nil {
		return 0, 0, false, false, err
	}

	return 0, copyPrefixUntil, !hasContentAfterOpenBracket, false, nil
}

func inspectMissingOpeningBracketPrefix(f *os.File, firstIdx, size int64) (copyPrefixUntil int64, isEmptyArray bool, err error) {
	searchEnd := size

	// If a trailing closing bracket exists, drop it first.
	if idx, b, found, err := findLastNonSpaceBefore(f, searchEnd); err != nil {
		return 0, false, err
	} else if !found {
		return firstIdx, true, nil
	} else if b == ']' {
		searchEnd = idx
	}

	for {
		idx, b, found, err := findLastNonSpaceBefore(f, searchEnd)
		if err != nil {
			return 0, false, err
		}
		if !found {
			return firstIdx, true, nil
		}

		if b == ',' {
			searchEnd = idx
			continue
		}

		if idx < firstIdx {
			return firstIdx, true, nil
		}

		return idx + 1, false, nil
	}
}

func findFirstNonSpaceInRange(f *os.File, start, end int64) (int64, byte, bool, error) {
	if start >= end {
		return 0, 0, false, nil
	}

	const chunkSize int64 = 4096
	buf := make([]byte, chunkSize)

	for offset := start; offset < end; {
		toRead := end - offset
		if toRead > chunkSize {
			toRead = chunkSize
		}

		n, err := f.ReadAt(buf[:toRead], offset)
		if err != nil && err != io.EOF {
			return 0, 0, false, err
		}

		for i := 0; i < n; i++ {
			if !isSpace(buf[i]) {
				return offset + int64(i), buf[i], true, nil
			}
		}

		offset += int64(n)
		if n == 0 {
			break
		}
	}

	return 0, 0, false, nil
}

func findLastNonSpaceBefore(f *os.File, end int64) (int64, byte, bool, error) {
	if end <= 0 {
		return 0, 0, false, nil
	}

	const chunkSize int64 = 4096
	buf := make([]byte, chunkSize)

	for right := end; right > 0; {
		left := right - chunkSize
		if left < 0 {
			left = 0
		}

		toRead := right - left
		n, err := f.ReadAt(buf[:toRead], left)
		if err != nil && err != io.EOF {
			return 0, 0, false, err
		}

		for i := n - 1; i >= 0; i-- {
			if !isSpace(buf[i]) {
				return left + int64(i), buf[i], true, nil
			}
		}

		right = left
		if n == 0 {
			break
		}
	}

	return 0, 0, false, nil
}

// isSpace is sufficient for JSON whitespace around the closing bracket.
func isSpace(b byte) bool {
	switch b {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}

// SaveTwitchLiveChatToFile connects to a Twitch channel and saves messages to a JSON file
func SaveTwitchLiveChatToFile(ctx context.Context, channel, filename string) error {
	const (
		maxRetries     = 10
		initialBackoff = 1 * time.Second
		maxBackoff     = 1 * time.Minute
	)

	retryCount := 0
	backoff := initialBackoff

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create client
		client := twitchIRC.NewAnonymousClient()

		// Channel for tracking connection errors
		errChan := make(chan error, 1)
		// Use a buffered notification channel and send on connect instead of
		// closing it. Closing from the callback could run multiple times and
		// cause a "close of closed channel" panic.
		connected := make(chan struct{}, 1)

		// Handle messages
		client.OnPrivateMessage(func(message twitchIRC.PrivateMessage) {
			comment := convertToLiveComment(message)
			if err := appendMessageToJSONArray(filename, comment); err != nil {
				log.Error().Err(err).Msg("error saving chat message")
			}
		})

		// Handle connection
		client.OnConnect(func() {
			log.Info().Msgf("connected to %s live chat room", channel)
			retryCount = 0
			backoff = initialBackoff
			select {
			case connected <- struct{}{}:
			default:
			}
		})

		client.Join(channel)

		// Connect in a goroutine so we can handle context cancellation
		go func() {
			if err := client.Connect(); err != nil {
				errChan <- err
			}
		}()

		// Wait for either connection success, error, or context cancellation
		select {
		case <-ctx.Done():
			err := client.Disconnect()
			if err != nil {
				log.Error().Err(err).Msg("error disconnecting from live chat")
			}
			return ctx.Err()
		case err := <-errChan:
			log.Error().Err(err).Msg("live chat connection error")
			retryCount++

			if retryCount >= maxRetries {
				return fmt.Errorf("max retries (%d) reached: %w", maxRetries, err)
			}

			log.Warn().Msgf("Retrying in %v (attempt %d/%d)...", backoff, retryCount, maxRetries)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		case <-connected:
			<-ctx.Done()
			log.Info().Msg("Context cancelled, disconnecting from live chat...")
			if err := client.Disconnect(); err != nil {
				log.Error().Err(err).Msg("error disconnecting from live chat")
			}
			return ctx.Err()
		}
	}
}
