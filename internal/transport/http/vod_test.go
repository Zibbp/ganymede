package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent/predicate"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

type searchVodServiceStub struct {
	VodService
	search func([]predicate.Vod) (vod.Pagination, error)
}

func (s searchVodServiceStub) SearchVods(_ context.Context, _, _ int, _ []utils.VodType, predicates []predicate.Vod, _ utils.VideoSort, _ utils.SortOrder) (vod.Pagination, error) {
	return s.search(predicates)
}

func TestSearchVodsNotes(t *testing.T) {
	t.Parallel()

	for _, fields := range []string{"notes", "title,notes"} {
		t.Run(fields, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			e.Validator = &utils.CustomValidator{Validator: validator.New()}
			called := false
			h := Handler{Service: Services{VodService: searchVodServiceStub{
				search: func(predicates []predicate.Vod) (vod.Pagination, error) {
					called = true
					require.Len(t, predicates, len(strings.Split(fields, ",")))
					selector := sql.Dialect(dialect.Postgres).Select().From(sql.Table("vods"))
					for _, p := range predicates {
						p(selector)
					}
					query, args := selector.Query()
					require.Contains(t, query, `"notes" ILIKE`)
					require.Contains(t, args, "%kept for tutorial%")
					if fields == "notes" {
						require.NotContains(t, query, `"title"`)
					} else {
						require.Contains(t, query, `"title" ILIKE`)
					}
					return vod.Pagination{}, nil
				},
			}}}
			req := httptest.NewRequest(http.MethodGet, "/vod/search?q=kept+for+tutorial&fields="+fields+"&limit=10&offset=0", nil)
			rec := httptest.NewRecorder()
			require.NoError(t, h.SearchVods(e.NewContext(req, rec)))
			require.Equal(t, http.StatusOK, rec.Code)
			require.True(t, called)
		})
	}
}

func TestGenerateThumbnailsVTTEscapesImagePath(t *testing.T) {
	t.Setenv("TWITCH_CLIENT_ID", "test")
	t.Setenv("TWITCH_CLIENT_SECRET", "test")
	t.Setenv("CDN_URL", "https://cdn.example.com")

	metadata := SpriteMetadata{
		Duration:       60,
		SpriteImages:   []string{"/videos/channel/my folder/sprite#001.jpg"},
		SpriteInterval: 60,
		SpriteRows:     1,
		SpriteColumns:  1,
		SpriteHeight:   124,
		SpriteWidth:    220,
	}

	vtt, err := GenerateThumbnailsVTT(metadata)
	if err != nil {
		t.Fatalf("GenerateThumbnailsVTT() unexpected error: %v", err)
	}

	expectedURL := "https://cdn.example.com/videos/channel/my%20folder/sprite%23001.jpg#xywh=0,0,220,124"
	if !strings.Contains(vtt, expectedURL) {
		t.Fatalf("expected escaped URL in VTT.\nexpected contains: %q\nactual: %q", expectedURL, vtt)
	}
}
