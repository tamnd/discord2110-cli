package discord2110

import (
	"context"
	"encoding/json"
	"fmt"
)

// Message is one Discord message record.
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Edited    string `json:"edited_timestamp"`
	Pinned    bool   `json:"pinned"`
	URL       string `json:"url"`
}

// FetchMessages fetches up to limit messages from a channel, paginating as needed.
func (c *Client) FetchMessages(ctx context.Context, channelID string, limit int) ([]*Message, error) {
	var msgs []*Message
	var before string

	for len(msgs) < limit {
		n := 100
		if limit-len(msgs) < 100 {
			n = limit - len(msgs)
		}
		path := fmt.Sprintf("/channels/%s/messages?limit=%d", channelID, n)
		if before != "" {
			path += "&before=" + before
		}
		batch, err := c.fetchMessageBatch(ctx, path, channelID)
		if err != nil {
			return msgs, err
		}
		if len(batch) == 0 {
			break
		}
		msgs = append(msgs, batch...)
		before = batch[len(batch)-1].ID
	}
	return msgs, nil
}

type rawMessage struct {
	ID              string `json:"id"`
	ChannelID       string `json:"channel_id"`
	Content         string `json:"content"`
	Timestamp       string `json:"timestamp"`
	EditedTimestamp string `json:"edited_timestamp"`
	Pinned          bool   `json:"pinned"`
	Author          struct {
		Username string `json:"username"`
	} `json:"author"`
}

func (c *Client) fetchMessageBatch(ctx context.Context, path, channelID string) ([]*Message, error) {
	body, err := c.request(ctx, path)
	if err != nil {
		return nil, err
	}
	var raw []rawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse messages: %w", err)
	}
	var out []*Message
	for _, m := range raw {
		cid := m.ChannelID
		if cid == "" {
			cid = channelID
		}
		out = append(out, &Message{
			ID:        m.ID,
			ChannelID: cid,
			Author:    m.Author.Username,
			Content:   m.Content,
			Timestamp: m.Timestamp,
			Edited:    m.EditedTimestamp,
			Pinned:    m.Pinned,
			URL:       fmt.Sprintf("https://discord.com/channels/@me/%s/%s", cid, m.ID),
		})
	}
	return out, nil
}
