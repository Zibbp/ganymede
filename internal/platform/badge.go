package platform

type Badges struct {
	Badges []Badge `json:"badges"`
}

type Badge struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	IamgeUrl    string `json:"image_url"`
	ImageUrl1X  string `json:"image_url_1x"`
	ImageUrl2X  string `json:"image_url_2x"`
	ImageUrl4X  string `json:"image_url_4x"`
	Description string `json:"description"`
	Title       string `json:"title"`
	ClickAction string `json:"click_action"`
	ClickUrl    string `json:"click_url"`
}
