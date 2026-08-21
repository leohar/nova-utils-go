// Package evm is a minimal JSON-RPC client for EVM chains, covering the
// calls needed to check node availability and sync state.
package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// LatestBlock returns the head block number reported by the node at url,
// via eth_getBlockByNumber("latest") — the same call the availability
// check times.
func LatestBlock(ctx context.Context, client *http.Client, url string) (uint64, error) {
	const request = `{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["latest",false]}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(request))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%s returned %s", url, resp.Status)
	}

	var body rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode response from %s: %w", url, err)
	}
	if body.Error != nil {
		return 0, fmt.Errorf("%s returned RPC error %d: %s", url, body.Error.Code, body.Error.Message)
	}

	var block struct {
		Number string `json:"number"`
	}
	if err := json.Unmarshal(body.Result, &block); err != nil || block.Number == "" {
		return 0, fmt.Errorf("%s returned no block: %s", url, body.Result)
	}
	return parseHexUint(block.Number)
}

func parseHexUint(s string) (uint64, error) {
	value, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parse block number %q: %w", s, err)
	}
	return value, nil
}
