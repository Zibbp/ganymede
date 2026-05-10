package utils_test

import (
	"testing"

	"github.com/zibbp/ganymede/internal/utils"
)

func TestApiKeyScope_Parse(t *testing.T) {
	cases := []struct {
		in       utils.ApiKeyScope
		wantOK   bool
		wantRes  utils.ApiKeyResource
		wantTier utils.ApiKeyTier
	}{
		{"vod:read", true, utils.ApiKeyResourceVod, utils.ApiKeyTierRead},
		{"playlist:write", true, utils.ApiKeyResourcePlaylist, utils.ApiKeyTierWrite},
		{"queue:admin", true, utils.ApiKeyResourceQueue, utils.ApiKeyTierAdmin},
		{"*:admin", true, utils.ApiKeyResourceWildcard, utils.ApiKeyTierAdmin},
		{"channel:read", true, utils.ApiKeyResourceChannel, utils.ApiKeyTierRead},
		// malformed
		{"", false, "", ""},
		{"vod", false, "", ""},
		{":read", false, "", ""},
		{"vod:", false, "", ""},
		{"unknown:read", false, "", ""},
		{"vod:bogus", false, "", ""},
		{"vod:read:extra", false, "", ""},
	}
	for _, tc := range cases {
		t.Run(string(tc.in), func(t *testing.T) {
			res, tier, ok := tc.in.Parse()
			if ok != tc.wantOK {
				t.Fatalf("Parse(%q) ok=%v, want %v", tc.in, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if res != tc.wantRes {
				t.Errorf("resource = %q, want %q", res, tc.wantRes)
			}
			if tier != tc.wantTier {
				t.Errorf("tier = %q, want %q", tier, tc.wantTier)
			}
		})
	}
}

func TestApiKeyScope_IsValid(t *testing.T) {
	for _, s := range utils.AllApiKeyScopes() {
		if !s.IsValid() {
			t.Errorf("AllApiKeyScopes contains invalid scope %q", s)
		}
	}
	for _, bad := range []utils.ApiKeyScope{"", "vod", "unknown:read", "vod:bogus"} {
		if bad.IsValid() {
			t.Errorf("scope %q should be invalid", bad)
		}
	}
}

func TestApiKeyScope_Includes(t *testing.T) {
	cases := []struct {
		holder   utils.ApiKeyScope
		required utils.ApiKeyScope
		want     bool
		why      string
	}{
		// Same resource, hierarchy.
		{"vod:admin", "vod:write", true, "admin includes write"},
		{"vod:admin", "vod:read", true, "admin includes read"},
		{"vod:write", "vod:read", true, "write includes read"},
		{"vod:read", "vod:read", true, "self-match"},
		{"vod:read", "vod:write", false, "read does not include write"},
		{"vod:write", "vod:admin", false, "write does not include admin"},

		// Cross-resource without wildcard: never matches.
		{"vod:admin", "playlist:read", false, "different resources"},
		{"playlist:write", "vod:write", false, "different resources, same tier"},

		// Wildcard holder satisfies any matching tier.
		{"*:admin", "vod:write", true, "wildcard admin covers vod:write"},
		{"*:admin", "playlist:admin", true, "wildcard admin covers playlist:admin"},
		{"*:write", "queue:read", true, "wildcard write covers queue:read"},
		{"*:read", "vod:write", false, "wildcard read does not include write"},

		// Required wildcard only satisfied by holder wildcard at same/higher tier.
		// (Useful for routes that say 'I require any-resource access' — currently
		// no route does this, but the semantics should be consistent.)
		{"*:admin", "*:write", true, "wildcard hierarchy"},
		{"vod:admin", "*:write", false, "specific resource cannot satisfy wildcard required"},

		// Invalid scopes never satisfy.
		{"vod:read", "bogus:read", false, "bogus required"},
		{"bogus:read", "vod:read", false, "bogus holder"},
	}
	for _, tc := range cases {
		t.Run(string(tc.holder)+"_includes_"+string(tc.required), func(t *testing.T) {
			got := tc.holder.Includes(tc.required)
			if got != tc.want {
				t.Errorf("(%q).Includes(%q) = %v, want %v (%s)", tc.holder, tc.required, got, tc.want, tc.why)
			}
		})
	}
}

func TestApiKeyScopes_Includes(t *testing.T) {
	scopes := utils.ApiKeyScopes{
		utils.ApiKeyScopeVodRead,
		utils.ApiKeyScopePlaylistWrite,
	}

	if !scopes.Includes(utils.ApiKeyScopeVodRead) {
		t.Error("scope list should include its exact element vod:read")
	}
	if !scopes.Includes(utils.ApiKeyScopePlaylistRead) {
		t.Error("playlist:write should include playlist:read")
	}
	if scopes.Includes(utils.ApiKeyScopeVodWrite) {
		t.Error("vod:read should NOT include vod:write")
	}
	if scopes.Includes(utils.ApiKeyScopeQueueRead) {
		t.Error("scope list with no queue scope should not include queue:read")
	}

	// Wildcard scope replaces the list.
	wildcard := utils.ApiKeyScopes{utils.ApiKeyScopeAllAdmin}
	for _, s := range utils.AllApiKeyScopes() {
		_, _, valid := s.Parse()
		if !valid {
			continue
		}
		// *:admin includes admin/write/read on every concrete resource.
		// It does NOT include `*:???` for tiers above admin (none exist).
		if !wildcard.Includes(s) {
			t.Errorf("*:admin should include %q", s)
		}
	}
}

func TestApiKeyScopesFromStrings_RoundTrip(t *testing.T) {
	in := []string{"vod:read", "queue:admin", "garbage"}
	scopes := utils.ApiKeyScopesFromStrings(in)
	if got := scopes.Strings(); !equalStrings(got, in) {
		t.Errorf("round trip mismatch: got %v, want %v", got, in)
	}
	// IsValid identifies the bad entry.
	if scopes[0].IsValid() != true || scopes[2].IsValid() != false {
		t.Errorf("expected first valid and third invalid, got valids %v / %v", scopes[0].IsValid(), scopes[2].IsValid())
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
