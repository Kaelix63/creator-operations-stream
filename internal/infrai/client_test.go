package infrai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCreateChannelRetriesWithSameIdentity(t *testing.T) {
	calls := 0
	var keys []string
	transport := roundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPost || req.URL.Path != "/v1/realtime/channel/create" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer auth")
		}
		keys = append(keys, req.Header.Get("Idempotency-Key"))
		status, body := http.StatusTooManyRequests, `{"ok":false,"data":null,"error":{"code":"busy","message":"retry"},"metadata":{}}`
		if calls == 2 {
			status, body = http.StatusOK, `{"ok":true,"data":{},"error":null,"metadata":{}}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	c := New("test-key")
	c.baseURL = "https://example.test"
	c.http = &http.Client{Transport: transport}
	c.sleep = func(time.Duration) {}
	if err := c.CreateChannel(context.Background(), "creator-operations"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || keys[0] == "" || keys[0] != keys[1] {
		t.Fatalf("calls=%d keys=%v", calls, keys)
	}
}
