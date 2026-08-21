package substrate_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leohar/nova-utils-go/chains"
	"github.com/leohar/nova-utils-go/substrate"
)

// Port of nova-utils tests/test_rpc_methods_availability.py.
//
// Every node of every Substrate chain in chains.json must expose the RPC
// methods the Nova Wallet clients rely on. Like the delayed asserts of the
// original, all missing methods of a node are reported at once — t.Errorf
// does not stop the subtest.

// requiredMethods is the list collected in the original test from
// substrate-sdk-android and nova-wallet-android usages; see the comments
// there for the excluded methods and why.
var requiredMethods = []string{
	"state_call", "state_getStorage", "state_subscribeStorage",
	"state_getKeysPaged", "state_getMetadata", "state_subscribeRuntimeVersion",
	"system_chain", "system_accountNextIndex", "system_properties",
	"chain_getBlock", "chain_getBlockHash", "chain_getHeader", "chain_getFinalizedHead",
	"childstate_getStorage",
	"author_submitExtrinsic", "author_submitAndWatchExtrinsic", "author_pendingExtrinsics",
}

// skippedNetworks mirrors tests/data/setting_data.py.
var skippedNetworks = map[string]bool{"Edgeware": true}

const requestTimeout = 20 * time.Second

func TestRPCMethodsAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test against live endpoints; skipped in -short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
	defer cancel()
	cfg, err := chains.LoadFromEnv(ctx)
	if err != nil {
		t.Fatalf("load chains config: %v", err)
	}

	substrateChains := chains.Substrate(cfg)
	if len(substrateChains) == 0 {
		t.Fatal("no Substrate chains in chains config")
	}

	for _, chain := range substrateChains {
		if skippedNetworks[chain.Name] {
			continue
		}
		t.Run(strings.ReplaceAll(chain.Name, "/", "_"), func(t *testing.T) {
			t.Parallel()
			for _, node := range chain.Nodes {
				t.Run(subtestName(node.URL), func(t *testing.T) {
					t.Parallel()
					ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
					defer cancel()

					client, err := substrate.Dial(ctx, node.URL)
					if err != nil {
						t.Fatalf("dial: %v", err)
					}
					defer client.Close()

					available, err := client.Methods(ctx)
					if err != nil {
						t.Fatalf("rpc_methods: %v", err)
					}
					for _, method := range requiredMethods {
						if !available[method] {
							t.Errorf("RPC method %s is not supported", method)
						}
					}
				})
			}
		})
	}
}

func subtestName(url string) string {
	name := strings.TrimPrefix(url, "wss://")
	name = strings.TrimPrefix(name, "ws://")
	return strings.ReplaceAll(strings.TrimSuffix(name, "/"), "/", "_")
}
