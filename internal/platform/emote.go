package platform

type Emotes struct {
	Emotes []Emote `json:"emotes"`
}

type Emote struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	URL    string      `json:"url"`
	Format EmoteFormat `json:"format"`
	Type   EmoteType   `json:"type"`
	Scale  string      `json:"scale"`
	Source string      `json:"source"`
	Width  int64       `json:"width"`
	Height int64       `json:"height"`
}

type EmoteFormat string

const (
	EmoteFormatStatic   EmoteFormat = "static"
	EmoteFormatAnimated EmoteFormat = "animated"
)

type EmoteType string

const (
	EmoteTypeGlobal       EmoteType = "global"
	EmoteTypeSubscription EmoteType = "subscription"
)
