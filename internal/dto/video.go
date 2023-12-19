package dto

import (
	"github.com/zibbp/ganymede/ent"
)

type ArchiveVideoInput struct {
	VideoID                   string
	Type                      string
	Platform                  string
	Resolution                string
	DownloadChat              bool
	RenderChat                bool
	Vod                       *ent.Vod
	Channel                   *ent.Channel
	Queue                     *ent.Queue
	LiveWatchChannel          *ent.Live
	LiveChatWorkflowId        string
	LiveChatArchiveWorkflowId string
}
