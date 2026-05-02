package api_key_test

import (
	"strings"
	"testing"

	"github.com/zibbp/ganymede/internal/api_key"
)

func TestGenerate_ProducesParseableToken(t *testing.T) {
	full, prefix, secret, err := api_key.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if !strings.HasPrefix(full, api_key.TokenBrand+"_") {
		t.Fatalf("expected token to start with %q, got %q", api_key.TokenBrand+"_", full)
	}
	gotPrefix, gotSecret, err := api_key.Parse(full)
	if err != nil {
		t.Fatalf("Parse(Generate()) error: %v", err)
	}
	if gotPrefix != prefix {
		t.Errorf("Parse prefix = %q, want %q", gotPrefix, prefix)
	}
	if gotSecret != secret {
		t.Errorf("Parse secret = %q, want %q", gotSecret, secret)
	}
}

func TestGenerate_UniqueTokens(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		full, _, _, err := api_key.Generate()
		if err != nil {
			t.Fatalf("Generate() error: %v", err)
		}
		if _, dup := seen[full]; dup {
			t.Fatalf("duplicate token generated at iteration %d: %s", i, full)
		}
		seen[full] = struct{}{}
	}
}

func TestHashAndVerify_RoundTrip(t *testing.T) {
	_, _, secret, err := api_key.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	hashed, err := api_key.HashSecret(secret)
	if err != nil {
		t.Fatalf("HashSecret() error: %v", err)
	}
	if hashed == secret {
		t.Fatal("HashSecret returned the plaintext secret")
	}
	if err := api_key.Verify(hashed, secret); err != nil {
		t.Errorf("Verify(matching) error: %v", err)
	}
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	hashed, err := api_key.HashSecret("right-secret")
	if err != nil {
		t.Fatalf("HashSecret() error: %v", err)
	}
	if err := api_key.Verify(hashed, "wrong-secret"); err == nil {
		t.Fatal("Verify(wrong-secret) returned nil error, want mismatch")
	}
}

func TestParse_RejectsMalformed(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"no underscores", "ganymedekey"},
		{"too few parts", "gym_abc"},
		{"too many parts", "gym_abc_def_ghi"},
		{"wrong brand", "github_abc123def456_secretvalue"},
		{"prefix wrong length", "gym_short_secretvalue"},
		{"prefix not hex", "gym_zzzzzzzzzzzz_secretvalue"},
		{"empty secret", "gym_abc123def456_"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := api_key.Parse(tc.in); err == nil {
				t.Fatalf("Parse(%q) returned nil error, want malformed", tc.in)
			}
		})
	}
}
