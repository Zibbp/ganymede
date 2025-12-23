package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	size := info.Size()

	// new file - write a complete array with a single element.
	if size == 0 {
		if _, err := f.Write([]byte("[\n")); err != nil {
			return fmt.Errorf("failed to write opening bracket: %w", err)
		}
		if _, err := f.Write(msg); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
		if _, err := f.Write([]byte("\n]\n")); err != nil {
			return fmt.Errorf("failed to write closing bracket: %w", err)
		}
		return f.Sync()
	}

	// Read only a small tail of the file to find the closing ']' and
	// determine whether the array is empty or not.
	const tailSize = 1024
	bufSize := size
	if bufSize > tailSize {
		bufSize = tailSize
	}

	buf := make([]byte, bufSize)
	if _, err := f.ReadAt(buf, size-bufSize); err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file tail: %w", err)
	}

	// Find last non-whitespace char (should be ']').
	i := int(bufSize - 1)
	for ; i >= 0 && isSpace(buf[i]); i-- {
	}
	if i < 0 || buf[i] != ']' {
		return fmt.Errorf("file %s is not a JSON array (missing closing ])", filename)
	}

	// Look backwards to see what’s before the closing ']' to check if array is empty.
	j := i - 1
	for ; j >= 0 && isSpace(buf[j]); j-- {
	}

	isEmptyArray := false
	if j >= 0 && buf[j] == '[' {
		isEmptyArray = true
	} else if size <= 2 {
		isEmptyArray = true
	}

	// Compute the absolute offset of the closing ']' in the file.
	lastBracketOffset := (size - bufSize) + int64(i)

	// Drop the closing ']' (and any trailing whitespace after it).
	if err := f.Truncate(lastBracketOffset); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	// Seek to the end after truncation.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// If the array already has elements, add a comma; otherwise just a newline.
	if isEmptyArray {
		if _, err := f.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	} else {
		if _, err := f.Write([]byte(",\n")); err != nil {
			return fmt.Errorf("failed to write comma: %w", err)
		}
	}

	// Write the new message and close the array again.
	if _, err := f.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if _, err := f.Write([]byte("\n]\n")); err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}

	return f.Sync()
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
