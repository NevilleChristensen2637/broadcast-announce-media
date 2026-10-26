package broadcast

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAnnouncementDecisionRequiresProcessedAsset(t *testing.T) {
	cases := []struct {
		name    string
		asset   Asset
		message string
		want    bool
	}{
		{"processed asset", Asset{ID: "a-1", Title: "Live set", Ready: true}, "Doors are open", true},
		{"unprocessed asset", Asset{ID: "a-2", Ready: false}, "Doors are open", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := announcementFor(tc.asset, tc.message)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPublishUsesEnvelopeAndRequestFields(t *testing.T) {
	var method, auth, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, auth = r.Method, r.Header.Get("Authorization")
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"data":{},"error":null,"metadata":{}}`))
	}))
	defer server.Close()
	if err := os.Setenv("INFRAI_API_KEY", "test-key"); err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Publish(context.Background(), "media-live", "acct-1", Announcement{AssetID: "a-1", Title: "Live set", Message: "Doors are open"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"channel", "event", "data", "account_id"} {
		if !strings.Contains(body, `"`+want+`"`) {
			t.Fatalf("request missing %s: %s", want, body)
		}
	}
	if method != http.MethodPost || auth != "Bearer test-key" {
		t.Fatalf("request boundary: method=%s auth=%s", method, auth)
	}
}
