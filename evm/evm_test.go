package evm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leohar/nova-utils-go/evm"
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

func TestLatestBlock(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    uint64
		wantErr bool
	}{
		{
			name:   "healthy node",
			status: http.StatusOK,
			body:   `{"jsonrpc":"2.0","id":1,"result":{"number":"0x158d3e2","hash":"0xabc"}}`,
			want:   0x158d3e2,
		},
		{
			name:    "http error",
			status:  http.StatusTooManyRequests,
			body:    `rate limited`,
			wantErr: true,
		},
		{
			name:    "rpc error",
			status:  http.StatusOK,
			body:    `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`,
			wantErr: true,
		},
		{
			name:    "null result",
			status:  http.StatusOK,
			body:    `{"jsonrpc":"2.0","id":1,"result":null}`,
			wantErr: true,
		},
		{
			name:    "malformed block number",
			status:  http.StatusOK,
			body:    `{"jsonrpc":"2.0","id":1,"result":{"number":"latest"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := serve(t, tt.status, tt.body)
			got, err := evm.LatestBlock(t.Context(), server.Client(), server.URL)
			if tt.wantErr {
				if err == nil {
					t.Fatal("LatestBlock succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("LatestBlock: %v", err)
			}
			if got != tt.want {
				t.Errorf("LatestBlock = %d, want %d", got, tt.want)
			}
		})
	}
}
