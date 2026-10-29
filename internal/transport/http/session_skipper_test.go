package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	session "github.com/canidam/echo-scs-session"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/database"
)

func TestSessionSkipper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		skip bool
	}{
		{"/_next/static/chunks/a.js", true},
		{"/", true},
		{"/vods/x", true},
		{"/health", true},
		{"/metrics", true},
		{"/swagger/index.html", true},
		{"/admin/api-keys", true},
		{"/riverui-settings", true},
		{"/apifoo", true},
		{"/api/v1/auth/me", false},
		{"/riverui", false},
		{"/riverui/", false},
		{"/riverui/api/jobs", false},
	}
	e := echo.New()
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, tc.path, nil), httptest.NewRecorder())
			assert.Equal(t, tc.skip, sessionSkipper(c))
		})
	}
}

// countingStore counts store round-trips; each is a DB query with pgxstore.
type countingStore struct {
	*memstore.MemStore
	finds   atomic.Int64
	commits atomic.Int64
}

func newCountingStore() *countingStore {
	return &countingStore{MemStore: memstore.NewWithCleanupInterval(0)}
}

func (s *countingStore) Find(token string) ([]byte, bool, error) {
	s.finds.Add(1)
	return s.MemStore.Find(token)
}

func (s *countingStore) Commit(token string, b []byte, expiry time.Time) error {
	s.commits.Add(1)
	return s.MemStore.Commit(token, b, expiry)
}

// newTestSessionManager mirrors production timeouts with a counting store.
// The default memstore from scs.New keeps one idle cleanup goroutine; calling
// StopCleanup right after construction races with its channel setup.
func newTestSessionManager(t *testing.T) (*scs.SessionManager, *countingStore) {
	t.Helper()
	sm := scs.New()
	store := newCountingStore()
	sm.Store = store
	sm.Lifetime = 30 * 24 * time.Hour
	sm.IdleTimeout = 7 * 24 * time.Hour
	return sm, store
}

// seedSession stores a session holding user_id and returns its token.
func seedSession(t *testing.T, sm *scs.SessionManager, store *countingStore) string {
	t.Helper()
	ctx, err := sm.Load(context.Background(), "")
	require.NoError(t, err)
	sm.Put(ctx, "user_id", "user-1")
	token, _, err := sm.Commit(ctx)
	require.NoError(t, err)
	store.finds.Store(0)
	store.commits.Store(0)
	return token
}

func TestSessionMiddlewareWithSession(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path    string
		skipped bool
	}{
		{"/_next/static/chunks/a.js", true},
		{"/", true},
		{"/health", true},
		{"/swagger/index.html", true},
		{"/api/v1/auth/me", false},
		{"/riverui/", false},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			sm, store := newTestSessionManager(t)
			token := seedSession(t, sm, store)

			var userID string
			e := echo.New()
			e.Use(session.LoadAndSaveWithConfig(session.Config{SessionManager: sm, Skipper: sessionSkipper}))
			e.Any("/*", func(c echo.Context) error {
				if !tc.skipped {
					userID = sm.GetString(c.Request().Context(), "user_id")
				}
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: "session", Value: token})
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			if tc.skipped {
				assert.Zero(t, store.finds.Load(), "finds")
				assert.Zero(t, store.commits.Load(), "commits")
				assert.Empty(t, rec.Header().Get("Set-Cookie"))
				return
			}
			assert.Equal(t, "user-1", userID)
			assert.EqualValues(t, 1, store.finds.Load(), "finds")
			// IdleTimeout refreshes expiry on every loaded request.
			assert.EqualValues(t, 1, store.commits.Load(), "commits")
			assert.Contains(t, rec.Header().Get("Set-Cookie"), "session="+token)
		})
	}
}

func TestSessionMiddlewareCreatesSession(t *testing.T) {
	t.Parallel()
	sm, store := newTestSessionManager(t)

	e := echo.New()
	e.Use(session.LoadAndSaveWithConfig(session.Config{SessionManager: sm, Skipper: sessionSkipper}))
	e.POST("/api/v1/auth/login", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := sm.RenewToken(ctx); err != nil {
			return err
		}
		sm.Put(ctx, "user_id", "user-1")
		return c.NoContent(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Zero(t, store.finds.Load(), "no cookie means no lookup")
	assert.EqualValues(t, 1, store.commits.Load())
	assert.Contains(t, rec.Header().Get("Set-Cookie"), "session=")
}

// TestNewHandlerSessionWiring guards the Skipper in NewHandler itself.
func TestNewHandlerSessionWiring(t *testing.T) {
	for k, v := range map[string]string{
		"DB_HOST": "127.0.0.1", "DB_PORT": "1", "DB_USER": "u", "DB_PASS": "p", "DB_NAME": "n",
		"TWITCH_CLIENT_ID": "x", "TWITCH_CLIENT_SECRET": "x",
		"VIDEOS_DIR": t.TempDir(), "TEMP_DIR": t.TempDir(),
	} {
		t.Setenv(k, v)
	}
	// pgxpool connects lazily; no request below reaches Postgres.
	pool, err := pgxpool.New(context.Background(), "postgres://u:p@127.0.0.1:1/n")
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	h := NewHandler(&database.Database{ConnPool: pool}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// The replaced pgxstore keeps an idle cleanup goroutine (5 min ticker);
	// StopCleanup here would race with its channel setup.
	store := newCountingStore()
	h.SessionManager.Store = store
	cookie := &http.Cookie{Name: "session", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}

	for _, p := range []string{"/health", "/swagger/index.html", "/riverui-settings"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		req.AddCookie(cookie)
		h.Server.ServeHTTP(httptest.NewRecorder(), req)
	}
	assert.Zero(t, store.finds.Load(), "unguarded routes must not load sessions")

	for _, p := range []string{"/api/v1/auth/me", "/riverui/"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		h.Server.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, p)
	}
	assert.EqualValues(t, 2, store.finds.Load(), "guarded routes must load sessions")
}
