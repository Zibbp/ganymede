package chapter

import (
	"testing"

	"github.com/zibbp/ganymede/ent"
)

func TestCreateWebVtt(t *testing.T) {
	t.Parallel()

	service := &Service{}

	t.Run("emits millisecond timestamps browsers can parse", func(t *testing.T) {
		t.Parallel()

		chapters := []*ent.Chapter{
			{Start: 0, End: 300, Title: "Intro"},
			{Start: 300, End: 2221, Title: "Diablo IV"},
		}

		got, err := service.CreateWebVtt(chapters)
		if err != nil {
			t.Fatalf("CreateWebVtt returned error: %v", err)
		}

		want := "WEBVTT\n\n" +
			"00:00:00.000 --> 00:05:00.000\nIntro\n\n" +
			"00:05:00.000 --> 00:37:01.000\nDiablo IV\n\n"

		if got != want {
			t.Errorf("CreateWebVtt output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("empty chapters yields header only", func(t *testing.T) {
		t.Parallel()

		got, err := service.CreateWebVtt(nil)
		if err != nil {
			t.Fatalf("CreateWebVtt returned error: %v", err)
		}

		if got != "WEBVTT\n\n" {
			t.Errorf("CreateWebVtt(nil) = %q, expected header only", got)
		}
	})
}
