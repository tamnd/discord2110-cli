package discord2110

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Server is the metadata for a Discord server (guild).
type Server struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon_url"`
	Banner      string   `json:"banner_url"`
	MemberCount int      `json:"member_count"`
	Online      int      `json:"online_count"`
	Verified    bool     `json:"verified"`
	NSFW        bool     `json:"nsfw"`
	Features    []string `json:"features"`
	InviteCode  string   `json:"invite_code"`
	URL         string   `json:"url"`
}

var invitePathRE = regexp.MustCompile(`(?:discord\.gg|discord\.com/invite)/([A-Za-z0-9\-]+)`)

// ExtractCode extracts a bare invite code from any format:
// bare code, discord.gg/code, or https://discord.com/invite/code.
func ExtractCode(input string) string {
	input = strings.TrimSpace(input)
	if m := invitePathRE.FindStringSubmatch(input); m != nil {
		return m[1]
	}
	// Bare code: no slashes, reasonable length
	if len(input) >= 4 && len(input) <= 30 && !strings.Contains(input, "/") {
		return input
	}
	return ""
}

// GetInvite resolves an invite code and returns the server metadata.
// No token is required for this endpoint.
func (c *Client) GetInvite(ctx context.Context, code string) (*Server, error) {
	code = ExtractCode(code)
	if code == "" {
		return nil, fmt.Errorf("could not extract invite code from %q", code)
	}
	path := fmt.Sprintf("/invites/%s?with_counts=true&with_expiration=true", code)
	body, err := c.request(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseInvite(body, code)
}

// inviteResponse is the raw Discord invite API response shape.
type inviteResponse struct {
	Guild struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Icon        string   `json:"icon"`
		Banner      string   `json:"banner"`
		Features    []string `json:"features"`
		NSFWLevel   int      `json:"nsfw_level"`
		Verified    bool     `json:"verified"`
	} `json:"guild"`
	ApproximateMemberCount   int `json:"approximate_member_count"`
	ApproximatePresenceCount int `json:"approximate_presence_count"`
}

func parseInvite(body []byte, code string) (*Server, error) {
	var inv inviteResponse
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, fmt.Errorf("parse invite: %w", err)
	}
	iconURL := ""
	if inv.Guild.Icon != "" {
		iconURL = fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.png", inv.Guild.ID, inv.Guild.Icon)
	}
	bannerURL := ""
	if inv.Guild.Banner != "" {
		bannerURL = fmt.Sprintf("https://cdn.discordapp.com/banners/%s/%s.png", inv.Guild.ID, inv.Guild.Banner)
	}
	return &Server{
		ID:          inv.Guild.ID,
		Name:        inv.Guild.Name,
		Description: inv.Guild.Description,
		Icon:        iconURL,
		Banner:      bannerURL,
		MemberCount: inv.ApproximateMemberCount,
		Online:      inv.ApproximatePresenceCount,
		Verified:    inv.Guild.Verified,
		NSFW:        inv.Guild.NSFWLevel > 0,
		Features:    inv.Guild.Features,
		InviteCode:  code,
		URL:         "https://discord.gg/" + code,
	}, nil
}
