package subquery_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leohar/nova-utils-go/subquery"
)

func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("request method = %s, want POST", r.Method)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestFetchMetadata(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    subquery.Metadata
		wantErr bool
	}{
		{
			name:   "synced project",
			status: http.StatusOK,
			body:   `{"data":{"_metadata":{"chain":"Polkadot","lastProcessedHeight":100,"targetHeight":103}}}`,
			want:   subquery.Metadata{Chain: "Polkadot", LastProcessedHeight: 100, TargetHeight: 103},
		},
		{
			name:    "http error",
			status:  http.StatusBadGateway,
			body:    `Bad Gateway`,
			wantErr: true,
		},
		{
			name:    "graphql error",
			status:  http.StatusOK,
			body:    `{"errors":[{"message":"project not found"}]}`,
			wantErr: true,
		},
		{
			name:    "missing metadata",
			status:  http.StatusOK,
			body:    `{"data":{}}`,
			wantErr: true,
		},
		{
			name:    "malformed json",
			status:  http.StatusOK,
			body:    `<html>maintenance</html>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := serve(t, tt.status, tt.body)
			got, err := subquery.FetchMetadata(t.Context(), server.Client(), server.URL)
			if tt.wantErr {
				if err == nil {
					t.Fatal("FetchMetadata succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchMetadata: %v", err)
			}
			if got != tt.want {
				t.Errorf("FetchMetadata = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFetchMetadataTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Timeout: 50 * time.Millisecond}
	if _, err := subquery.FetchMetadata(t.Context(), client, server.URL); err == nil {
		t.Fatal("FetchMetadata succeeded, want timeout error")
	}
}

func TestMetadataLag(t *testing.T) {
	behind := subquery.Metadata{LastProcessedHeight: 90, TargetHeight: 100}
	if got := behind.Lag(); got != 10 {
		t.Errorf("Lag = %d, want 10", got)
	}
	ahead := subquery.Metadata{LastProcessedHeight: 100, TargetHeight: 98}
	if got := ahead.Lag(); got != 2 {
		t.Errorf("Lag = %d, want 2", got)
	}
}
