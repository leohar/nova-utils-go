// Package subquery is a minimal client for the SubQuery indexer GraphQL API,
// covering the _metadata endpoint used to check indexing progress.
package subquery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Metadata is the indexing progress reported by a SubQuery project.
type Metadata struct {
	Chain               string `json:"chain"`
	LastProcessedHeight int64  `json:"lastProcessedHeight"`
	TargetHeight        int64  `json:"targetHeight"`
}

// Lag returns how many blocks the indexer is behind the chain.
func (m Metadata) Lag() int64 {
	lag := m.TargetHeight - m.LastProcessedHeight
	if lag < 0 {
		lag = -lag
	}
	return lag
}

const metadataQuery = `{"query":"query { _metadata { chain lastProcessedHeight targetHeight } }"}`

type graphqlResponse struct {
	Data struct {
		Metadata *Metadata `json:"_metadata"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchMetadata queries the _metadata object of the SubQuery project at url.
func FetchMetadata(ctx context.Context, client *http.Client, url string) (Metadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(metadataQuery))
	if err != nil {
		return Metadata{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return Metadata{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Metadata{}, fmt.Errorf("%s returned %s", url, resp.Status)
	}

	var body graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Metadata{}, fmt.Errorf("decode response from %s: %w", url, err)
	}
	if len(body.Errors) > 0 {
		return Metadata{}, fmt.Errorf("%s returned GraphQL error: %s", url, body.Errors[0].Message)
	}
	if body.Data.Metadata == nil {
		return Metadata{}, fmt.Errorf("%s returned no _metadata object", url)
	}
	return *body.Data.Metadata, nil
}
