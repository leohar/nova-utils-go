package substrate_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"

	"github.com/leohar/nova-utils-go/substrate"
)

// wsServer starts a WebSocket server that, for each received request,
// writes the corresponding group of frames in order.
func wsServer(t *testing.T, exchanges ...[]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		for _, frames := range exchanges {
			if _, _, err := conn.Read(r.Context()); err != nil {
				return
			}
			for _, frame := range frames {
				if err := conn.Write(r.Context(), websocket.MessageText, []byte(frame)); err != nil {
					return
				}
			}
		}
		// Keep the connection open until the client is done.
		conn.Read(r.Context())
	}))
	t.Cleanup(server.Close)
	return server
}

func dial(t *testing.T, url string) *substrate.Client {
	t.Helper()
	client, err := substrate.Dial(t.Context(), url)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestMethods(t *testing.T) {
	server := wsServer(t, []string{
		`{"jsonrpc":"2.0","id":1,"result":{"methods":["state_call","system_chain"],"version":1}}`,
	})

	got, err := dial(t, server.URL).Methods(t.Context())
	if err != nil {
		t.Fatalf("Methods: %v", err)
	}
	if !got["state_call"] || !got["system_chain"] || len(got) != 2 {
		t.Errorf("Methods = %v, want state_call and system_chain", got)
	}
}

func TestCallRPCError(t *testing.T) {
	server := wsServer(t, []string{
		`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
	})

	if err := dial(t, server.URL).Call(t.Context(), "no_such_method", []any{}, nil); err == nil {
		t.Fatal("Call succeeded, want RPC error")
	}
}

func TestCallSkipsUnrelatedMessages(t *testing.T) {
	// A subscription notification (no id) and a stale response (foreign id)
	// arrive before the answer to our request; both must be skipped.
	server := wsServer(t, []string{
		`{"jsonrpc":"2.0","method":"state_storage","params":{"subscription":"abc","result":{}}}`,
		`{"jsonrpc":"2.0","id":99,"result":"stale"}`,
		`{"jsonrpc":"2.0","id":1,"result":"Polkadot"}`,
	})

	var result string
	if err := dial(t, server.URL).Call(t.Context(), "system_chain", []any{}, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != "Polkadot" {
		t.Errorf("Call result = %q, want Polkadot", result)
	}
}

func TestCallSequentialIDs(t *testing.T) {
	server := wsServer(t,
		[]string{`{"jsonrpc":"2.0","id":1,"result":"first"}`},
		[]string{`{"jsonrpc":"2.0","id":2,"result":"second"}`},
	)
	client := dial(t, server.URL)

	var first, second string
	if err := client.Call(t.Context(), "system_chain", []any{}, &first); err != nil {
		t.Fatalf("first Call: %v", err)
	}
	if err := client.Call(t.Context(), "system_chain", []any{}, &second); err != nil {
		t.Fatalf("second Call: %v", err)
	}
	if first != "first" || second != "second" {
		t.Errorf("Call results = %q, %q; want first, second", first, second)
	}
}
