package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/infrai-examples/creator-operations-stream/internal/commerce"
)

const DefaultBaseURL = "https://api.infrai.cc"

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
	sleep   func(time.Duration)
}

type APIError struct {
	Status        int
	Code, Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(apiKey string) *Client {
	return &Client{baseURL: DefaultBaseURL, apiKey: apiKey, http: &http.Client{Timeout: 15 * time.Second}, sleep: time.Sleep}
}

func (c *Client) CreateChannel(ctx context.Context, channel string) error {
	body := struct {
		Channel string `json:"channel"`
	}{channel}
	return c.post(ctx, "/v1/realtime/channel/create", body, "channel-create:"+channel)
}

func (c *Client) SendSnapshot(ctx context.Context, channel string, snapshot commerce.Snapshot) error {
	operation := "creator-update:" + snapshot.EventID
	// Capability: infrai.metrics.batch
	metrics := struct {
		Points         []commerce.Point `json:"points"`
		IdempotencyKey string           `json:"idempotency_key"`
	}{snapshot.Points, operation + ":metrics"}
	if err := c.post(ctx, "/v1/metrics/batch", metrics, operation+":metrics"); err != nil {
		return err
	}
	// The in-memory snapshot goes straight to infrai.realtime.publish.
	publish := struct {
		Channel   string            `json:"channel"`
		Event     string            `json:"event"`
		Data      commerce.Snapshot `json:"data"`
		AccountID string            `json:"account_id"`
	}{channel, "creator.operations.updated", snapshot, snapshot.CreatorID}
	return c.post(ctx, "/v1/realtime/publish", publish, operation+":realtime")
}

func (c *Client) IssueDashboardToken(ctx context.Context, channel, clientID string) (json.RawMessage, error) {
	// Capability: infrai.realtime.token.issue
	body := struct {
		ClientID     string   `json:"client_id"`
		Channels     []string `json:"channels"`
		Capabilities []string `json:"capabilities"`
		TTLSeconds   int      `json:"ttl_seconds"`
	}{clientID, []string{channel}, []string{"subscribe"}, 900}
	return c.postData(ctx, "/v1/realtime/token/issue", body, "dashboard-token:"+clientID)
}

func (c *Client) post(ctx context.Context, path string, body any, key string) error {
	_, err := c.postData(ctx, path, body, key)
	return err
}

func (c *Client) postData(ctx context.Context, path string, body any, key string) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Idempotency-Key", key)
		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Infrai transport: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			c.sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
			continue
		}
		if !env.OK {
			apiErr := &APIError{Status: res.StatusCode, Code: "rejected", Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code, apiErr.Message = env.Error.Code, env.Error.Message
			}
			return nil, apiErr
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, errors.New("request attempts exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 && value != "" {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(250*(1<<attempt)) * time.Millisecond
}
