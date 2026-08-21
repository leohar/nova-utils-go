package evm_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leohar/nova-utils-go/chains"
	"github.com/leohar/nova-utils-go/evm"
)

// Port of nova-utils tests/test_eth_nodes_availability.py.
//
// Every HTTPS RPC node of Ethereum mainnet (eip155:1) must answer
// eth_getBlockByNumber("latest") within maxLatency, and its head must be
// within maxLagBlocks of the best head observed across the nodes.
//
// The Python original compares each node against a hardcoded Infura
// endpoint (with an API key embedded in the test). Here all nodes are
// probed concurrently at the same moment and compared against each other,
// so no external reference node or credential is needed.
const (
	maxLatency     = 3 * time.Second
	maxLagBlocks   = 3
	requestTimeout = 15 * time.Second
)

func TestEthNodesAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test against live endpoints; skipped in -short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
	defer cancel()
	cfg, err := chains.LoadFromEnv(ctx)
	if err != nil {
		t.Fatalf("load chains config: %v", err)
	}

	ethereum := chains.ByChainID(cfg, "eip155:1")
	if ethereum == nil {
		t.Fatal("no eip155:1 chain in chains config")
	}
	nodes := ethereum.HTTPSNodes()
	if len(nodes) == 0 {
		t.Fatal("no HTTPS nodes for Ethereum in chains config")
	}

	// Probe all nodes concurrently so their heads are sampled at the same
	// moment and can be compared against each other.
	type probe struct {
		height  uint64
		latency time.Duration
		err     error
	}
	client := &http.Client{}
	results := make([]probe, len(nodes))
	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
			defer cancel()
			start := time.Now()
			height, err := evm.LatestBlock(ctx, client, node.URL)
			results[i] = probe{height: height, latency: time.Since(start), err: err}
		}()
	}
	wg.Wait()

	var bestHead uint64
	for _, r := range results {
		if r.err == nil && r.height > bestHead {
			bestHead = r.height
		}
	}

	for i, node := range nodes {
		t.Run(subtestName(node.URL), func(t *testing.T) {
			r := results[i]
			if r.err != nil {
				t.Fatalf("eth_getBlockByNumber: %v", r.err)
			}
			t.Logf("head %d, latency %s", r.height, r.latency)
			if r.latency > maxLatency {
				t.Errorf("request took %s, more than %s", r.latency, maxLatency)
			}
			if lag := bestHead - r.height; lag > maxLagBlocks {
				t.Errorf("node is %d blocks behind the best observed head: node=%d, best=%d",
					lag, r.height, bestHead)
			}
		})
	}
}

func subtestName(url string) string {
	name := strings.TrimPrefix(url, "https://")
	name = strings.TrimPrefix(name, "http://")
	return strings.ReplaceAll(strings.TrimSuffix(name, "/"), "/", "_")
}
