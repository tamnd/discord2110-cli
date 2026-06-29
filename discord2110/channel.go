package discord2110

import (
	"context"
	"encoding/json"
	"fmt"
)

// Channel is one Discord channel record.
type Channel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	TypeName string `json:"type_name"`
	Topic    string `json:"topic"`
	Position int    `json:"position"`
	NSFW     bool   `json:"nsfw"`
	ServerID string `json:"server_id"`
	URL      string `json:"url"`
}

// ChannelTypeName returns the human-readable name for a Discord channel type.
func ChannelTypeName(t int) string {
	switch t {
	case 0:
		return "text"
	case 2:
		return "voice"
	case 4:
		return "category"
	case 5:
		return "announcement"
	case 13:
		return "stage"
	case 15:
		return "forum"
	default:
		return "unknown"
	}
}

// IsTextChannel reports whether a channel type supports readable messages.
func IsTextChannel(t int) bool {
	return t == 0 || t == 5 || t == 15
}

// GetChannels fetches the channels for a guild. Requires token.
func (c *Client) GetChannels(ctx context.Context, guildID string) ([]*Channel, error) {
	path := fmt.Sprintf("/guilds/%s/channels", guildID)
	body, err := c.request(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseChannels(body, guildID)
}

// rawChannel is the raw Discord channel object shape.
type rawChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Topic    string `json:"topic"`
	Position int    `json:"position"`
	NSFW     bool   `json:"nsfw"`
	GuildID  string `json:"guild_id"`
}

func parseChannels(body []byte, guildID string) ([]*Channel, error) {
	var raw []rawChannel
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse channels: %w", err)
	}
	var out []*Channel
	for _, ch := range raw {
		sid := ch.GuildID
		if sid == "" {
			sid = guildID
		}
		out = append(out, &Channel{
			ID:       ch.ID,
			Name:     ch.Name,
			Type:     ch.Type,
			TypeName: ChannelTypeName(ch.Type),
			Topic:    ch.Topic,
			Position: ch.Position,
			NSFW:     ch.NSFW,
			ServerID: sid,
			URL:      fmt.Sprintf("https://discord.com/channels/%s/%s", sid, ch.ID),
		})
	}
	return out, nil
}
