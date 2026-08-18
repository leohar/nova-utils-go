// Package chains loads the Nova Wallet chains configuration
// (https://github.com/novasamatech/nova-utils) and provides helpers
// for querying it.
package chains

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// DefaultURL points to the latest chains config served over raw GitHub,
// the same way Nova Wallet clients consume it.
const DefaultURL = "https://raw.githubusercontent.com/novasamatech/nova-utils/master/chains/v22/chains.json"

// Chain is a single network entry of chains.json. Only the fields needed
// by the tests are mapped; the config carries more.
type Chain struct {
	ChainID     string                   `json:"chainId"`
	Name        string                   `json:"name"`
	Assets      []Asset                  `json:"assets"`
	Nodes       []Node                   `json:"nodes"`
	ExternalAPI map[string][]ExternalAPI `json:"externalApi"`
	Options     []string                 `json:"options"`
}

type Asset struct {
	Symbol    string `json:"symbol"`
	Precision int    `json:"precision"`
}

type Node struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type ExternalAPI struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Load decodes a chains.json document.
func Load(r io.Reader) ([]Chain, error) {
	var chains []Chain
	if err := json.NewDecoder(r).Decode(&chains); err != nil {
		return nil, fmt.Errorf("decode chains config: %w", err)
	}
	return chains, nil
}

// LoadFile reads a chains.json document from a local file.
func LoadFile(path string) ([]Chain, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}

// Fetch downloads a chains.json document from url.
func Fetch(ctx context.Context, url string) ([]Chain, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch chains config: %s returned %s", url, resp.Status)
	}
	return Load(resp.Body)
}

// LoadFromEnv loads the chains config the way the test suite expects:
// from the local file named by CHAINS_JSON_PATH if set, otherwise over
// HTTP from CHAINS_JSON_URL or DefaultURL.
func LoadFromEnv(ctx context.Context) ([]Chain, error) {
	if path := os.Getenv("CHAINS_JSON_PATH"); path != "" {
		return LoadFile(path)
	}
	url := os.Getenv("CHAINS_JSON_URL")
	if url == "" {
		url = DefaultURL
	}
	return Fetch(ctx, url)
}

// ByChainID returns the chain with the given chainId, or nil.
func ByChainID(chains []Chain, id string) *Chain {
	for i := range chains {
		if chains[i].ChainID == id {
			return &chains[i]
		}
	}
	return nil
}

// HasOption reports whether the chain's options list contains name.
func (c Chain) HasOption(name string) bool {
	for _, option := range c.Options {
		if option == name {
			return true
		}
	}
	return false
}

// HTTPSNodes returns the chain's nodes reachable over HTTPS,
// skipping WebSocket-only endpoints.
func (c Chain) HTTPSNodes() []Node {
	var nodes []Node
	for _, node := range c.Nodes {
		if strings.HasPrefix(node.URL, "https://") {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// Substrate returns the chains that expose a Substrate runtime: entries
// marked PAUSED in their name or carrying the noSubstrateRuntime option
// are dropped, mirroring nova-utils tests/data/setting_data.py.
func Substrate(chains []Chain) []Chain {
	var result []Chain
	for _, chain := range chains {
		if strings.Contains(chain.Name, "PAUSED") || chain.HasOption("noSubstrateRuntime") {
			continue
		}
		result = append(result, chain)
	}
	return result
}

// SubqueryURLs returns the deduplicated, sorted list of SubQuery endpoints
// referenced by any externalApi section of the given chains.
func SubqueryURLs(chains []Chain) []string {
	seen := make(map[string]struct{})
	for _, chain := range chains {
		for _, apis := range chain.ExternalAPI {
			for _, api := range apis {
				if api.Type == "subquery" && api.URL != "" {
					seen[api.URL] = struct{}{}
				}
			}
		}
	}
	urls := make([]string, 0, len(seen))
	for url := range seen {
		urls = append(urls, url)
	}
	sort.Strings(urls)
	return urls
}
