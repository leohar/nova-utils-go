# nova-utils-go

Go port of the [nova-utils](https://github.com/novasamatech/nova-utils) integration
test suite — the tests that validate Nova Wallet's chains configuration against
live infrastructure (RPC nodes, indexers).

The original suite is written in Python/pytest and runs mostly sequentially.
This project ports it test by test to idiomatic Go: typed config parsing,
`t.Run` subtests with `t.Parallel()`, `context` timeouts, and offline unit
tests alongside the live integration checks.

## Ported tests

| Original (pytest) | Go | Notes |
|---|---|---|
| `test_subquery_is_synced.py` | `subquery/sync_test.go` | For every SubQuery endpoint in `chains.json`, asserts the indexer is < 10 blocks behind the chain head. All endpoints are checked concurrently. Unlike the original, an endpoint that fails to respond **fails the test** instead of being silently skipped. |

Planned next: Ethereum RPC node availability, Substrate RPC method availability,
node sync checks.

## Layout

- `chains/` — types and loaders for `chains.json`, the network config that
  Nova Wallet clients consume over raw GitHub.
- `subquery/` — minimal client for the SubQuery `_metadata` GraphQL endpoint,
  plus the sync integration test.

No dependencies outside the Go standard library.

## Running

Offline unit tests only:

```sh
go test -short ./...
```

Full suite, including live integration checks (fetches the latest
`chains.json` from raw GitHub and queries every SubQuery endpoint in it):

```sh
go test ./...
```

Configuration via environment variables:

| Variable | Effect |
|---|---|
| `CHAINS_JSON_PATH` | Read the chains config from a local file instead of the network. |
| `CHAINS_JSON_URL` | Fetch the chains config from a custom URL. |

Example against a local nova-utils checkout:

```sh
CHAINS_JSON_PATH=../nova-utils/chains/v22/chains.json go test ./...
```

## CI

Pull requests run `go vet` and the offline unit tests. The live integration
suite runs on a daily schedule and on manual dispatch — its failures are
findings about the config or infrastructure, not about the code under review.
