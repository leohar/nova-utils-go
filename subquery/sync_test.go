package subquery_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/leohar/nova-utils-go/chains"
	"github.com/leohar/nova-utils-go/subquery"
)

// Port of nova-utils tests/test_subquery_is_synced.py.
//
// For every SubQuery endpoint referenced in chains.json the test asserts that
// the indexer is at most maxLag blocks behind the chain head. Unlike the
// Python original, an endpoint that fails to respond is a test failure,
// not a silent pass.
const (
	maxLag         = 10
	requestTimeout = 15 * time.Second
)

func TestSubqueryIsSynced(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test against live endpoints; skipped in -short mode")
	}

	urls := chains.SubqueryURLs(loadChains(t))
	if len(urls) == 0 {
		t.Fatal("no SubQuery endpoints found in chains config")
	}
	t.Logf("checking %d SubQuery endpoints", len(urls))

	client := &http.Client{}
	for _, url := range urls {
		t.Run(subtestName(url), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
			defer cancel()

			md, err := subquery.FetchMetadata(ctx, client, url)
			if err != nil {
				t.Fatalf("fetch _metadata: %v", err)
			}
			if lag := md.Lag(); lag >= maxLag {
				t.Errorf("indexer for %s is %d blocks behind the chain: lastProcessedHeight=%d, targetHeight=%d",
					md.Chain, lag, md.LastProcessedHeight, md.TargetHeight)
			}
		})
	}
}

func loadChains(t *testing.T) []chains.Chain {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
	defer cancel()
	cfg, err := chains.LoadFromEnv(ctx)
	if err != nil {
		t.Fatalf("load chains config: %v", err)
	}
	return cfg
}

// subtestName turns an endpoint URL into a flat subtest name:
// slashes would otherwise create nested subtests in go test output.
func subtestName(url string) string {
	name := strings.TrimPrefix(url, "https://")
	name = strings.TrimPrefix(name, "http://")
	return strings.ReplaceAll(strings.TrimSuffix(name, "/"), "/", "_")
}
