package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

// TestGetTwitchVideoQualityOptions tests the GetTwitchVideoQualityOptions function
func TestGetTwitchVideoQualityOptions(t *testing.T) {
	testCases := []struct {
		videoId     string
		expectedLen int
		description string
		videoType   utils.VodType
		channelName string
	}{
		{
			videoId:     "2325332129",
			expectedLen: 9,
			description: "Standard archive VOD",
			videoType:   utils.Archive,
			channelName: "datmodz",
		},
		{
			videoId:     "CleverPolishedSwordPanicBasket-qmNOWICct4rtR_wX",
			expectedLen: 6,
			description: "Clip",
			videoType:   utils.Clip,
			channelName: "datmodz",
		}}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Setup VOD object
			v := ent.Vod{
				ExtID: tc.videoId,
				Type:  tc.videoType,
			}
			c := ent.Channel{}
			v.Edges.Channel = &c
			v.Edges.Channel.Name = tc.channelName

			qualities, err := GetTwitchVideoQualityOptions(context.Background(), v)
			assert.NoError(t, err)

			t.Logf("Qualities for VOD %s: %v", tc.videoId, qualities)

			assert.Equal(t, tc.expectedLen, len(qualities),
				"Expected %d quality options for VOD %s, got %d",
				tc.expectedLen, tc.videoId, len(qualities))
		})
	}
}
