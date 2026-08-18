# nova-utils-go

Go port of the [nova-utils](https://github.com/novasamatech/nova-utils) integration
test suite — the tests that validate Nova Wallet's chains configuration against
live infrastructure (RPC nodes, indexers).

The original suite is written in Python/pytest and runs mostly sequentially.
This project ports it test by test to idiomatic Go: typed config parsing,
`t.Run` subtests with `t.Parallel()`, `context` timeouts, and offline unit
tests alongside the live integration checks.

Tests are ported one pull request at a time — see the PR history for
per-test notes and live run reports.
