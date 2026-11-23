package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMediaRoutePrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"production absolute", "/data/videos", "/data/videos"},
		{"development relative", "dev/videos", "/dev/videos"},
		{"development dot relative", "./dev/videos", "/dev/videos"},
		{"trailing slash", "/data/videos/", "/data/videos"},
		{"repeated leading slashes", "//data/videos", "/data/videos"},
		{"parent relative", "../videos", "/videos"},
		{"nested parent", "dev/../videos", "/videos"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, mediaRoutePrefix(tc.dir))
		})
	}
}
