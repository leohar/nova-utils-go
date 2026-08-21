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
