package chains_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leohar/nova-utils-go/chains"
)

const sampleConfig = `[
  {
    "chainId": "91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3",
    "name": "Polkadot",
    "assets": [{"symbol": "DOT", "precision": 10}],
    "nodes": [{"url": "wss://rpc.polkadot.io", "name": "Parity node"}],
    "externalApi": {
      "history": [
        {"type": "subquery", "url": "https://history.example.org"},
        {"type": "etherscan", "url": "https://etherscan.example.org"}
      ],
      "governance": [
        {"type": "subquery", "url": "https://governance.example.org"}
      ]
    }
  },
  {
    "chainId": "b0a8d493285c2df73290dfb7e61f870f17b41801197a149ca93654499ea3dafe",
    "name": "Kusama",
    "externalApi": {
      "history": [
        {"type": "subquery", "url": "https://history.example.org"}
      ]
    }
  },
  {
    "chainId": "eip155:1",
    "name": "Ethereum"
  }
]`

func TestLoad(t *testing.T) {
	got, err := chains.Load(strings.NewReader(sampleConfig))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Load returned %d chains, want 3", len(got))
	}

	polkadot := got[0]
	if polkadot.Name != "Polkadot" {
		t.Errorf("first chain name = %q, want Polkadot", polkadot.Name)
	}
	if len(polkadot.Assets) != 1 || polkadot.Assets[0].Precision != 10 {
		t.Errorf("unexpected Polkadot assets: %+v", polkadot.Assets)
	}
	if len(polkadot.ExternalAPI["history"]) != 2 {
		t.Errorf("unexpected Polkadot history APIs: %+v", polkadot.ExternalAPI["history"])
	}
}

func TestSubqueryURLs(t *testing.T) {
	cfg, err := chains.Load(strings.NewReader(sampleConfig))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := chains.SubqueryURLs(cfg)
	// Deduplicated across chains and API groups, non-subquery types dropped, sorted.
	want := []string{
		"https://governance.example.org",
		"https://history.example.org",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SubqueryURLs = %v, want %v", got, want)
	}
}
