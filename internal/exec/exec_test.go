package exec

import (
	osExec "os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestStartArchiveCommand(t *testing.T) {
	t.Parallel()

	cmd := osExec.Command("sh", "-c", "exit 0")
	cmd.SysProcAttr = vodArchiveProcessAttributes()

	done, err := startArchiveCommand(cmd)
	if err != nil {
		t.Fatalf("start archive command: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait for archive command: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for archive command")
	}
}

func TestLiveArchiveProcessAttributes(t *testing.T) {
	t.Parallel()

	attrs := liveArchiveProcessAttributes()
	if !attrs.Setpgid {
		t.Fatal("live archive process must run in its own process group")
	}
	if attrs.Pdeathsig != syscall.SIGINT {
		t.Fatalf("parent death signal = %v, want SIGINT", attrs.Pdeathsig)
	}
}

func TestVodArchiveProcessAttributes(t *testing.T) {
	t.Parallel()

	attrs := vodArchiveProcessAttributes()
	if !attrs.Setpgid {
		t.Fatal("VOD archive process must run in its own process group")
	}
	if attrs.Pdeathsig != syscall.SIGTERM {
		t.Fatalf("parent death signal = %v, want SIGTERM", attrs.Pdeathsig)
	}
}

func Test_extractSharedChatArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "empty",
			in:   nil,
			want: nil,
		},
		{
			name: "no shared flags",
			in:   []string{"-h", "1440", "-w", "340", "--font", "Inter"},
			want: nil,
		},
		{
			name: "equals form",
			in:   []string{"-h", "1440", "--stv=false", "--font", "Inter"},
			want: []string{"--stv=false"},
		},
		{
			name: "space form",
			in:   []string{"--bttv", "false", "-h", "1440"},
			want: []string{"--bttv", "false"},
		},
		{
			name: "all three providers mixed forms",
			in:   []string{"--framerate", "30", "--bttv=true", "--ffz", "false", "--stv=false"},
			want: []string{"--bttv=true", "--ffz", "false", "--stv=false"},
		},
		{
			name: "temp-path space form",
			in:   []string{"-h", "1440", "--temp-path", "/var/cache/td"},
			want: []string{"--temp-path", "/var/cache/td"},
		},
		{
			name: "temp-path equals form",
			in:   []string{"--temp-path=/var/cache/td", "--font", "Inter"},
			want: []string{"--temp-path=/var/cache/td"},
		},
		{
			name: "trailing flag without value",
			in:   []string{"--stv"},
			want: []string{"--stv"},
		},
		{
			name: "bare boolean does not swallow following flag",
			in:   []string{"--stv", "--temp-path", "/var/cache/td"},
			want: []string{"--stv", "--temp-path", "/var/cache/td"},
		},
		{
			name: "does not match prefix-only flags",
			in:   []string{"--stvthing", "--bttvfoo=1", "--temp-pathish"},
			want: nil,
		},
		{
			name: "collision is intentionally not forwarded",
			in:   []string{"--collision", "rename", "--stv=false"},
			want: []string{"--stv=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSharedChatArgs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractSharedChatArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func Test_appendFFmpegLiveOutputStreamArgs(t *testing.T) {
	tests := []struct {
		name      string
		audioOnly bool
		want      []string
	}{
		{
			name:      "all streams",
			audioOnly: false,
			want:      []string{"-map", "0", "-dn", "-ignore_unknown", "-c", "copy"},
		},
		{
			name:      "audio only",
			audioOnly: true,
			want:      []string{"-map", "0:a", "-dn", "-ignore_unknown", "-c", "copy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendFFmpegLiveOutputStreamArgs(nil, tt.audioOnly)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendFFmpegLiveOutputStreamArgs(nil, %t) = %v, want %v", tt.audioOnly, got, tt.want)
			}
		})
	}
}
