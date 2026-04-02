package exec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	twitchIRC "github.com/gempir/go-twitch-irc/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	liveChatPendingFileSuffix = ".pending.ndjson"
	liveChatSyncInterval      = 2 * time.Second
	liveChatMainFlushInterval = 5 * time.Second
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

type liveChatJSONArrayWriter struct {
	filename    string
	pendingPath string
	pendingFile *os.File

	mu              sync.Mutex
	lastPendingSync time.Time
}

func newLiveChatJSONArrayWriter(filename string) (*liveChatJSONArrayWriter, error) {
	if err := ensureJSONArrayFileExists(filename); err != nil {
		return nil, err
	}

	if err := RecoverTwitchLiveChatPendingFile(filename); err != nil {
		return nil, err
	}

	pendingPath := filename + liveChatPendingFileSuffix

	pendingFile, err := os.OpenFile(pendingPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open pending chat file: %w", err)
	}

	return &liveChatJSONArrayWriter{
		filename:        filename,
		pendingPath:     pendingPath,
		pendingFile:     pendingFile,
		lastPendingSync: time.Now(),
	}, nil
}

func ensureJSONArrayFileExists(filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat chat file: %w", err)
	}

	if err := os.WriteFile(filename, []byte("[]\n"), 0o644); err != nil {
		return fmt.Errorf("failed to initialize chat file: %w", err)
	}

	return nil
}

// RecoverTwitchLiveChatPendingFile merges any crash-left pending live chat
// messages into the main JSON file, keeping the main file valid JSON.
func RecoverTwitchLiveChatPendingFile(filename string) error {
	pendingPath := filename + liveChatPendingFileSuffix

	mainExists := true
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			mainExists = false
		} else {
			return fmt.Errorf("failed to stat chat file: %w", err)
		}
	}

	pendingExists := true
	if _, err := os.Stat(pendingPath); err != nil {
		if os.IsNotExist(err) {
			pendingExists = false
		} else {
			return fmt.Errorf("failed to stat pending chat file: %w", err)
		}
	}

	if !mainExists && !pendingExists {
		return nil
	}

	if !mainExists {
		if err := ensureJSONArrayFileExists(filename); err != nil {
			return err
		}
	}

	mergedCount, skippedInvalidCount, err := mergePendingNDJSONIntoJSONArray(filename, pendingPath)
	if err != nil {
		return fmt.Errorf("failed to recover pending chat messages: %w", err)
	}

	if err := os.Remove(pendingPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove recovered pending chat file: %w", err)
	}

	if mergedCount > 0 || skippedInvalidCount > 0 {
		log.Debug().
			Str("chat_file", filename).
			Int("recovered_messages", mergedCount).
			Int("skipped_invalid_pending_messages", skippedInvalidCount).
			Msg("recovered pending live chat messages")
	}

	return nil
}

func (w *liveChatJSONArrayWriter) Append(comment utils.LiveComment) error {
	// Encode the new comment once.
	msg, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pendingFile == nil {
		return fmt.Errorf("cannot append: pending file is finalized")
	}

	if _, err := w.pendingFile.Write(msg); err != nil {
		return fmt.Errorf("failed to append pending chat message: %w", err)
	}

	if w.pendingFile == nil {
		return fmt.Errorf("cannot append: pending file is finalized")
	}
	if _, err := w.pendingFile.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to append pending chat newline: %w", err)
	}

	if time.Since(w.lastPendingSync) >= liveChatSyncInterval {
		if err := w.pendingFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync pending chat file: %w", err)
		}
		w.lastPendingSync = time.Now()
	}

	return nil
}

func (w *liveChatJSONArrayWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.flushLocked()
}

func (w *liveChatJSONArrayWriter) flushLocked() error {
	if w.pendingFile == nil {
		return nil
	}

	if err := w.pendingFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync pending chat file before flush: %w", err)
	}

	info, err := w.pendingFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat pending chat file before flush: %w", err)
	}

	if info.Size() == 0 {
		w.lastPendingSync = time.Now()
		return nil
	}

	if err := w.pendingFile.Close(); err != nil {
		return fmt.Errorf("failed to close pending chat file before flush: %w", err)
	}
	w.pendingFile = nil

	mergedCount, skippedInvalidCount, err := mergePendingNDJSONIntoJSONArray(w.filename, w.pendingPath)
	if err != nil {
		// Best effort reopen so ingestion can continue if merge failed.
		reopenErr := w.reopenPendingFile(false)
		if reopenErr != nil {
			return fmt.Errorf("failed to merge pending chat into primary JSON: %w; also failed to reopen pending chat file: %v", err, reopenErr)
		}

		return fmt.Errorf("failed to merge pending chat into primary JSON: %w", err)
	}

	if err := w.reopenPendingFile(true); err != nil {
		return err
	}
	w.lastPendingSync = time.Now()

	if mergedCount > 0 || skippedInvalidCount > 0 {
		log.Debug().
			Str("chat_file", w.filename).
			Int("merged_messages", mergedCount).
			Int("skipped_invalid_pending_messages", skippedInvalidCount).
			Msg("flushed pending live chat messages")
	}

	return nil
}

func (w *liveChatJSONArrayWriter) reopenPendingFile(truncate bool) error {
	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if truncate {
		flags |= os.O_TRUNC
	}

	pendingFile, err := os.OpenFile(w.pendingPath, flags, 0o644)
	if err != nil {
		return fmt.Errorf("failed to reopen pending chat file: %w", err)
	}
	w.pendingFile = pendingFile

	return nil
}

func (w *liveChatJSONArrayWriter) Finalize() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pendingFile == nil {
		return nil
	}

	if err := w.flushLocked(); err != nil {
		return err
	}

	if err := w.pendingFile.Close(); err != nil {
		return fmt.Errorf("failed to close pending chat file before finalize: %w", err)
	}
	w.pendingFile = nil

	if err := os.Remove(w.pendingPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pending chat file: %w", err)
	}

	return nil
}

func mergePendingNDJSONIntoJSONArray(filename, pendingPath string) (mergedCount, skippedInvalidCount int, err error) {
	pf, err := os.Open(pendingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("failed to open pending chat file: %w", err)
	}
	defer pf.Close() //nolint:errcheck

	// Open (or create) the destination JSON file for read/write.
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	// Acquire an exclusive advisory lock so only one finalizer mutates file state.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return 0, 0, fmt.Errorf("failed to lock file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}()

	info, err := f.Stat()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to stat file: %w", err)
	}

	copyFrom, copyPrefixUntil, isEmptyArray, needsOpeningBracket, err := inspectJSONArrayPrefix(f, info.Size())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to inspect existing JSON array: %w", err)
	}

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := f.Seek(copyFrom, io.SeekStart); err != nil {
		return 0, 0, fmt.Errorf("failed to seek original file: %w", err)
	}

	if needsOpeningBracket {
		if _, err := tmp.Write([]byte("[\n")); err != nil {
			return 0, 0, fmt.Errorf("failed to write opening bracket: %w", err)
		}
	}

	if copyPrefixUntil > copyFrom {
		if _, err := io.CopyN(tmp, f, copyPrefixUntil-copyFrom); err != nil {
			return 0, 0, fmt.Errorf("failed to copy existing JSON prefix: %w", err)
		}
	}

	hasAnyElement := !isEmptyArray
	reader := bufio.NewReader(pf)

	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) > 0 {
				if json.Valid(trimmed) {
					if hasAnyElement {
						if _, err := tmp.Write([]byte(",\n")); err != nil {
							return mergedCount, skippedInvalidCount, fmt.Errorf("failed to write message separator: %w", err)
						}
					} else {
						if _, err := tmp.Write([]byte("\n")); err != nil {
							return mergedCount, skippedInvalidCount, fmt.Errorf("failed to write newline: %w", err)
						}
						hasAnyElement = true
					}

					if _, err := tmp.Write(trimmed); err != nil {
						return mergedCount, skippedInvalidCount, fmt.Errorf("failed to write pending chat message: %w", err)
					}
					mergedCount++
				} else {
					skippedInvalidCount++
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return mergedCount, skippedInvalidCount, fmt.Errorf("failed reading pending chat file: %w", readErr)
		}
	}

	if !hasAnyElement {
		if _, err := tmp.Write([]byte("\n")); err != nil {
			return mergedCount, skippedInvalidCount, fmt.Errorf("failed to write empty array newline: %w", err)
		}
	}

	if _, err := tmp.Write([]byte("\n]\n")); err != nil {
		return mergedCount, skippedInvalidCount, fmt.Errorf("failed to write closing bracket: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return mergedCount, skippedInvalidCount, fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return mergedCount, skippedInvalidCount, fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return mergedCount, skippedInvalidCount, fmt.Errorf("failed to atomically replace file: %w", err)
	}

	// Best-effort sync of parent directory to increase rename durability.
	if dirF, err := os.Open(dir); err == nil {
		_ = dirF.Sync()
		_ = dirF.Close()
	}

	return mergedCount, skippedInvalidCount, nil
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
		statusInterval = 2 * time.Minute
		idleWarnAfter  = 10 * time.Minute
	)

	retryCount := 0
	backoff := initialBackoff

	for {
		attemptStartedAt := time.Now()
		logger := log.With().Str("channel", channel).Str("chat_file", filename).Logger()
		logger.Debug().Msg("starting live chat connection attempt")

		chatWriter, err := newLiveChatJSONArrayWriter(filename)
		if err != nil {
			return fmt.Errorf("failed to initialize live chat writer: %w", err)
		}

		finalizeWriter := func() {
			if err := chatWriter.Finalize(); err != nil {
				logger.Error().Err(err).Msg("failed to finalize live chat writer")
			}
		}

		select {
		case <-ctx.Done():
			finalizeWriter()
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

		var messagesSaved atomic.Int64
		var writeErrors atomic.Int64
		var lastMessageReceivedUnixNano atomic.Int64
		lastMessageReceivedUnixNano.Store(time.Now().UnixNano())

		// Handle messages
		client.OnPrivateMessage(func(message twitchIRC.PrivateMessage) {
			receivedAt := time.Now()
			lastMessageReceivedUnixNano.Store(receivedAt.UnixNano())

			comment := convertToLiveComment(message)
			writeStarted := time.Now()
			if err := chatWriter.Append(comment); err != nil {
				errors := writeErrors.Add(1)
				logger.Error().Err(err).
					Int64("message_write_errors", errors).
					Str("message_id", message.ID).
					Msg("error saving chat message")
				return
			}

			writeDuration := time.Since(writeStarted)
			count := messagesSaved.Add(1)

			if writeDuration > 2*time.Second {
				logger.Warn().
					Dur("write_duration", writeDuration).
					Int64("messages_saved", count).
					Msg("slow chat message write")
			}

			if count == 1 || count%500 == 0 {
				logger.Debug().
					Int64("messages_saved", count).
					Dur("message_age", receivedAt.Sub(message.Time)).
					Time("twitch_message_time", message.Time).
					Msg("live chat message saved")
			}
		})

		// Handle connection
		client.OnConnect(func() {
			logger.Info().
				Dur("connection_attempt_elapsed", time.Since(attemptStartedAt)).
				Msgf("connected to %s live chat room", channel)
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

		connectedOnce := false
		shouldRetry := false
		statusTicker := time.NewTicker(statusInterval)
		flushTicker := time.NewTicker(liveChatMainFlushInterval)
		stopTickers := func() {
			statusTicker.Stop()
			flushTicker.Stop()
		}

		for {
			select {
			case <-ctx.Done():
				logger.Info().
					Int64("messages_saved", messagesSaved.Load()).
					Int64("message_write_errors", writeErrors.Load()).
					Dur("run_elapsed", time.Since(attemptStartedAt)).
					Msg("Context cancelled, disconnecting from live chat...")
				stopTickers()
				if err := client.Disconnect(); err != nil {
					logger.Error().Err(err).Msg("error disconnecting from live chat")
				}
				finalizeWriter()
				return ctx.Err()
			case <-connected:
				connectedOnce = true
				logger.Debug().Msg("live chat connection marked as established")
			case <-flushTicker.C:
				if err := chatWriter.Flush(); err != nil {
					errors := writeErrors.Add(1)
					logger.Error().Err(err).
						Int64("message_write_errors", errors).
						Msg("error flushing pending live chat messages to primary JSON")
				}
			case <-statusTicker.C:
				lastMsgAt := time.Unix(0, lastMessageReceivedUnixNano.Load())
				idleFor := time.Since(lastMsgAt)

				event := logger.Debug()
				if connectedOnce && idleFor > idleWarnAfter {
					event = logger.Warn()
				}

				event.
					Bool("connected_once", connectedOnce).
					Int64("messages_saved", messagesSaved.Load()).
					Int64("message_write_errors", writeErrors.Load()).
					Dur("idle_for", idleFor).
					Dur("run_elapsed", time.Since(attemptStartedAt)).
					Msg("live chat ingest status")
			case err := <-errChan:
				if ctx.Err() != nil {
					stopTickers()
					finalizeWriter()
					return ctx.Err()
				}

				if connectedOnce {
					logger.Warn().Err(err).
						Int64("messages_saved", messagesSaved.Load()).
						Int64("message_write_errors", writeErrors.Load()).
						Dur("run_elapsed", time.Since(attemptStartedAt)).
						Msg("live chat disconnected")
				} else {
					logger.Error().Err(err).
						Dur("run_elapsed", time.Since(attemptStartedAt)).
						Msg("live chat connection error")
				}

				retryCount++
				if retryCount >= maxRetries {
					stopTickers()
					finalizeWriter()
					return fmt.Errorf("max retries (%d) reached: %w", maxRetries, err)
				}

				logger.Warn().Msgf("Retrying in %v (attempt %d/%d)...", backoff, retryCount, maxRetries)

				if disErr := client.Disconnect(); disErr != nil {
					logger.Error().Err(disErr).Msg("error disconnecting from live chat")
				}

				finalizeWriter()

				select {
				case <-ctx.Done():
					stopTickers()
					return ctx.Err()
				case <-time.After(backoff):
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}

				shouldRetry = true
			}

			if shouldRetry {
				stopTickers()
				break
			}
		}
	}
}
