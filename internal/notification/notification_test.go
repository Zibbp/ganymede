package notification

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	entNotification "github.com/zibbp/ganymede/ent/notification"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestRedactURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "valid webhook url redacts path and query",
			raw:  "https://discord.com/api/webhooks/123/secret?token=abc",
			want: "https://discord.com/***",
		},
		{
			name: "valid url without path still redacted",
			raw:  "http://example.com",
			want: "http://example.com/***",
		},
		{
			name: "invalid short input returns generic redaction",
			raw:  "invalid",
			want: "***",
		},
		{
			name: "invalid long input returns truncated redaction",
			raw:  "this-is-not-a-valid-url",
			want: "this-is-not-***",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redactURL(tc.raw)
			if got != tc.want {
				t.Fatalf("redactURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	t.Parallel()

	t.Run("replaces known variables and preserves unknown", func(t *testing.T) {
		t.Parallel()

		got := renderTemplate(
			"Hello {{ channel_display_name }} - task={{failed_task}} - missing={{unknown}}",
			map[string]interface{}{
				"channel_display_name": "Demo Channel",
				"failed_task":          "video_download",
			},
		)

		want := "Hello Demo Channel - task=video_download - missing={{unknown}}"
		if got != want {
			t.Fatalf("renderTemplate() = %q, want %q", got, want)
		}
	})

	t.Run("nil value leaves placeholder untouched", func(t *testing.T) {
		t.Parallel()

		got := renderTemplate("value={{x}}", map[string]interface{}{"x": nil})
		want := "value={{x}}"
		if got != want {
			t.Fatalf("renderTemplate() = %q, want %q", got, want)
		}
	})

	t.Run("single-pass behavior does not recursively process replacements", func(t *testing.T) {
		t.Parallel()

		got := renderTemplate(
			"x={{a}} y={{b}}",
			map[string]interface{}{
				"a": "{{b}}",
				"b": "done",
			},
		)

		want := "x={{b}} y=done"
		if got != want {
			t.Fatalf("renderTemplate() = %q, want %q", got, want)
		}
	})
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	t.Run("zero time becomes empty string", func(t *testing.T) {
		t.Parallel()
		if got := formatTime(time.Time{}); got != "" {
			t.Fatalf("formatTime(zero) = %q, want empty string", got)
		}
	})

	t.Run("non-zero time formatted RFC3339", func(t *testing.T) {
		t.Parallel()

		ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		got := formatTime(ts)
		want := "2026-01-02T03:04:05Z"
		if got != want {
			t.Fatalf("formatTime() = %q, want %q", got, want)
		}
	})
}

func TestGetVariableMap(t *testing.T) {
	t.Parallel()

	t.Run("returns complete defaults when entities are nil", func(t *testing.T) {
		t.Parallel()

		m := getVariableMap(nil, nil, nil, "", nil)

		for _, key := range []string{
			"failed_task", "category", "channel_id", "channel_ext_id", "channel_display_name",
			"vod_id", "vod_ext_id", "vod_platform", "vod_type", "vod_title", "vod_duration",
			"vod_views", "vod_resolution", "vod_streamed_at", "vod_created_at", "queue_id", "queue_created_at",
		} {
			if _, ok := m[key]; !ok {
				t.Fatalf("expected key %q to exist", key)
			}
		}

		if m["vod_duration"] != 0 {
			t.Fatalf("expected default vod_duration=0, got %#v", m["vod_duration"])
		}
		if m["vod_views"] != 0 {
			t.Fatalf("expected default vod_views=0, got %#v", m["vod_views"])
		}
		if m["vod_created_at"] != "" {
			t.Fatalf("expected default vod_created_at empty, got %#v", m["vod_created_at"])
		}
	})

	t.Run("fills values from provided entities", func(t *testing.T) {
		t.Parallel()

		channelID := uuid.New()
		vodID := uuid.New()
		queueID := uuid.New()
		streamedAt := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
		createdAt := time.Date(2025, 5, 2, 10, 0, 0, 0, time.UTC)
		queueCreatedAt := time.Date(2025, 5, 3, 10, 0, 0, 0, time.UTC)
		category := "Gaming"

		m := getVariableMap(
			&ent.Channel{ID: channelID, ExtID: "chan-ext", DisplayName: "Channel Name"},
			&ent.Vod{
				ID:         vodID,
				ExtID:      "vod-ext",
				Platform:   utils.VideoPlatform("twitch"),
				Type:       utils.VodType("archive"),
				Title:      "Title",
				Duration:   123,
				Views:      456,
				Resolution: "1080p60",
				StreamedAt: streamedAt,
				CreatedAt:  createdAt,
			},
			&ent.Queue{ID: queueID, CreatedAt: queueCreatedAt},
			"video_download",
			&category,
		)

		if m["channel_id"] != channelID {
			t.Fatalf("unexpected channel_id: %#v", m["channel_id"])
		}
		if m["vod_id"] != vodID {
			t.Fatalf("unexpected vod_id: %#v", m["vod_id"])
		}
		if m["queue_id"] != queueID {
			t.Fatalf("unexpected queue_id: %#v", m["queue_id"])
		}
		if m["vod_streamed_at"] != streamedAt.Format(time.RFC3339) {
			t.Fatalf("unexpected vod_streamed_at: %#v", m["vod_streamed_at"])
		}
		if m["queue_created_at"] != queueCreatedAt.Format(time.RFC3339) {
			t.Fatalf("unexpected queue_created_at: %#v", m["queue_created_at"])
		}
		if m["failed_task"] != "video_download" {
			t.Fatalf("unexpected failed_task: %#v", m["failed_task"])
		}
		if m["category"] != "Gaming" {
			t.Fatalf("unexpected category: %#v", m["category"])
		}
	})
}

func TestGetTestVariableMap(t *testing.T) {
	t.Parallel()

	m := getTestVariableMap()

	for _, key := range []string{
		"channel_id", "vod_id", "queue_id", "vod_streamed_at", "vod_created_at", "queue_created_at", "failed_task", "category",
	} {
		if _, ok := m[key]; !ok {
			t.Fatalf("expected key %q to exist", key)
		}
	}

	if m["failed_task"] != "" {
		t.Fatalf("expected failed_task empty, got %#v", m["failed_task"])
	}
	if m["category"] != "" {
		t.Fatalf("expected category empty, got %#v", m["category"])
	}

	for _, key := range []string{"vod_streamed_at", "vod_created_at", "queue_created_at"} {
		value, ok := m[key].(string)
		if !ok || value == "" {
			t.Fatalf("expected %q to be non-empty RFC3339 string, got %#v", key, m[key])
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			t.Fatalf("expected %q to be RFC3339, got %q (err=%v)", key, value, err)
		}
	}
}

func TestSendUnknownType(t *testing.T) {
	t.Parallel()

	s := &Service{}
	err := s.send(context.Background(), &ent.Notification{Type: entNotification.Type("invalid")}, "body", nil)
	if err == nil {
		t.Fatal("expected error for unknown notification provider type")
	}
	if !strings.Contains(err.Error(), "unknown notification provider type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendTestNotificationUnknownEventType(t *testing.T) {
	t.Parallel()

	s := &Service{}
	err := s.SendTestNotification(context.Background(), &ent.Notification{}, "not-an-event")
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
	if !strings.Contains(err.Error(), "unknown test notification event type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
