package broadcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Asset struct {
	ID        string
	Title     string
	Ready     bool
	CreatorID string
}
type Announcement struct {
	AssetID string `json:"asset_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Client struct {
	baseURL, key string
	http         *http.Client
	sleep        func(time.Duration)
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	key := strings.TrimSpace(os.Getenv("INFRAI_API_KEY"))
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{strings.TrimRight(baseURL, "/"), key, httpClient, time.Sleep}, nil
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("infrai response rejected: status=%d body=%s", e.Status, e.Body)
}

func (c *Client) call(ctx context.Context, method, path string, input any, output any, requestID string) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, strings.NewReader(string(payload)))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if seconds, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			c.sleep(delay)
			continue
		}
		if !env.OK {
			return &APIError{Status: res.StatusCode, Body: string(env.Error)}
		}
		if output != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, output); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
		if res.StatusCode >= 400 {
			return &APIError{Status: res.StatusCode, Body: string(body)}
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

type presence struct {
	Members []struct {
		ID string `json:"id"`
	} `json:"members"`
}

func (c *Client) MemberIDs(ctx context.Context, channel string) ([]string, error) {
	var p presence
	if err := c.call(ctx, http.MethodGet, "/v1/realtime/presence/get/"+channel, nil, &p, "presence-"+channel); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(p.Members))
	for _, member := range p.Members {
		ids = append(ids, member.ID)
	}
	return ids, nil
}

type publishRequest struct {
	Channel   string          `json:"channel"`
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	AccountID string          `json:"account_id"`
}

func (c *Client) Publish(ctx context.Context, channel, accountID string, message Announcement) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	// POST /v1/realtime/publish is the delivery boundary.
	return c.call(ctx, http.MethodPost, "/v1/realtime/publish", publishRequest{channel, "media.announcement", data, accountID}, nil, "announcement-"+message.AssetID)
}

func announcementFor(asset Asset, message string) (Announcement, bool) {
	if strings.TrimSpace(asset.ID) == "" || !asset.Ready || strings.TrimSpace(message) == "" {
		return Announcement{}, false
	}
	return Announcement{AssetID: asset.ID, Title: asset.Title, Message: message}, true
}
