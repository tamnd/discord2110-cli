package discord2110_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/any-cli/kit/errs"
	. "github.com/tamnd/discord2110-cli/discord2110"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 0
	cfg.MaxWait = 1 // tiny max wait so rate-limit test triggers immediately
	return NewClient(cfg), srv
}

func isRateLimited(err error) bool { return errs.KindOf(err) == errs.KindRateLimited }
func isNotFound(err error) bool    { return errs.KindOf(err) == errs.KindNotFound }

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseURL == "" {
		t.Error("BaseURL is empty")
	}
	if cfg.UserAgent == "" {
		t.Error("UserAgent is empty")
	}
	if cfg.Rate <= 0 {
		t.Error("Rate must be > 0")
	}
	if cfg.MaxWait <= 0 {
		t.Error("MaxWait must be > 0")
	}
}

func TestNewClient(t *testing.T) {
	if c := NewClient(DefaultConfig()); c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestExtractCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ggTWnwK", "ggTWnwK"},
		{"https://discord.gg/ggTWnwK", "ggTWnwK"},
		{"discord.com/invite/abc123", "abc123"},
		{"https://discord.com/invite/abc123", "abc123"},
		{"", ""},
	}
	for _, tc := range cases {
		got := ExtractCode(tc.in)
		if got != tc.want {
			t.Errorf("ExtractCode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

const inviteJSON = `{
  "guild": {
    "id": "102860784329052160",
    "name": "Reactiflux",
    "description": "React community",
    "icon": "abc123",
    "banner": "",
    "features": ["COMMUNITY"],
    "nsfw_level": 0,
    "verified": true
  },
  "approximate_member_count": 180000,
  "approximate_presence_count": 4500
}`

func TestGetInvite_OK(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(inviteJSON))
	})
	defer srv.Close()

	srv2, err := c.GetInvite(context.Background(), "ggTWnwK")
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if srv2.Name != "Reactiflux" {
		t.Errorf("Name = %q, want Reactiflux", srv2.Name)
	}
	if srv2.MemberCount != 180000 {
		t.Errorf("MemberCount = %d, want 180000", srv2.MemberCount)
	}
	if !srv2.Verified {
		t.Error("Verified should be true")
	}
	if srv2.InviteCode != "ggTWnwK" {
		t.Errorf("InviteCode = %q, want ggTWnwK", srv2.InviteCode)
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Unknown Invite", "code": 10006}`))
	})
	defer srv.Close()

	_, err := c.GetInvite(context.Background(), "invalid")
	if !isNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
	}
}

func TestGetInvite_Blocked_401(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "401: Unauthorized"}`))
	})
	defer srv.Close()

	_, err := c.GetInvite(context.Background(), "ggTWnwK")
	if !isRateLimited(err) {
		t.Errorf("expected RateLimited (blocked 401), got: %v", err)
	}
}

func TestGetInvite_RateLimit_Exceeded(t *testing.T) {
	// retry_after of 60s exceeds MaxWait of 1ns (set in newTestClient)
	body, _ := json.Marshal(map[string]any{"message": "rate limited", "retry_after": 60.0})
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(body)
	})
	defer srv.Close()

	_, err := c.GetInvite(context.Background(), "ggTWnwK")
	if !isRateLimited(err) {
		t.Errorf("expected RateLimited, got: %v", err)
	}
}

func TestIsTextChannel(t *testing.T) {
	cases := []struct {
		t    int
		want bool
	}{
		{0, true},
		{2, false},
		{4, false},
		{5, true},
		{13, false},
		{15, true},
		{99, false},
	}
	for _, tc := range cases {
		got := IsTextChannel(tc.t)
		if got != tc.want {
			t.Errorf("IsTextChannel(%d) = %v, want %v", tc.t, got, tc.want)
		}
	}
}

func TestChannelTypeName(t *testing.T) {
	cases := []struct {
		t    int
		want string
	}{
		{0, "text"},
		{2, "voice"},
		{4, "category"},
		{5, "announcement"},
		{13, "stage"},
		{15, "forum"},
		{99, "unknown"},
	}
	for _, tc := range cases {
		got := ChannelTypeName(tc.t)
		if got != tc.want {
			t.Errorf("ChannelTypeName(%d) = %q, want %q", tc.t, got, tc.want)
		}
	}
}
