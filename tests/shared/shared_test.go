package tests_shared

import (
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
)

func TestRiverJobBelongsToArchive(t *testing.T) {
	videoID := uuid.New()
	queueID := uuid.New()

	tests := []struct {
		name string
		job  *rivertype.JobRow
		want bool
	}{
		{
			name: "video argument",
			job:  &rivertype.JobRow{EncodedArgs: []byte(`{"video_id":"` + videoID.String() + `"}`)},
			want: true,
		},
		{
			name: "legacy video argument",
			job:  &rivertype.JobRow{EncodedArgs: []byte(`{"VideoID":"` + videoID.String() + `"}`)},
			want: true,
		},
		{
			name: "queue argument",
			job:  &rivertype.JobRow{EncodedArgs: []byte(`{"input":{"queue_id":"` + queueID.String() + `"}}`)},
			want: true,
		},
		{
			name: "queue metadata",
			job:  &rivertype.JobRow{Metadata: []byte(`{"ganymede":{"queue_id":"` + queueID.String() + `"}}`)},
			want: true,
		},
		{
			name: "unrelated periodic job",
			job:  &rivertype.JobRow{EncodedArgs: []byte(`{}`), Metadata: []byte(`{}`)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := riverJobBelongsToArchive(tt.job, videoID, queueID); got != tt.want {
				t.Fatalf("riverJobBelongsToArchive() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestRiverJobBlocksArchiveCompletion(t *testing.T) {
	videoID := uuid.New()
	queueID := uuid.New()
	queueArgs := []byte(`{"input":{"queue_id":"` + queueID.String() + `"}}`)

	tests := []struct {
		name string
		job  *rivertype.JobRow
		want bool
	}{
		{
			name: "archive pipeline job",
			job:  &rivertype.JobRow{Kind: "post_process_video", EncodedArgs: queueArgs},
			want: true,
		},
		{
			name: "post-archive stream ID enrichment",
			job:  &rivertype.JobRow{Kind: postArchiveStreamVideoIDJobKind, EncodedArgs: queueArgs},
			want: false,
		},
		{
			name: "unrelated active job",
			job:  &rivertype.JobRow{Kind: "periodic_cleanup", EncodedArgs: []byte(`{}`)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := riverJobBlocksArchiveCompletion(tt.job, videoID, queueID); got != tt.want {
				t.Fatalf("riverJobBlocksArchiveCompletion() = %t, want %t", got, tt.want)
			}
		})
	}
}
