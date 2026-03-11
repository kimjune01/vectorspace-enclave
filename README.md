# VectorSpace Enclave

Standalone TEE auction package for [AWS Nitro Enclaves](https://aws.amazon.com/ec2/nitro/nitro-enclaves/). Runs a VCG ad auction inside an attested enclave so the exchange operator never sees the user's query embedding.

**Attestation requires a standalone repo.** An auditor clones this repo, runs `go build`, hashes the binary, and compares it to the attestation document. No ambiguity about what code is inside the enclave.

## How It Works

1. Publisher SDK encrypts a query embedding with the enclave's attested public key
2. Enclave decrypts the embedding, loads positions and budgets (pushed by the exchange)
3. Filters by budget and optional τ threshold
4. Scores: `log₅(price) - dist²/σ²`
5. Picks a winner, computes VCG payment
6. Zeros the embedding, returns `{winner_id, payment}`

The exchange only sees who won and what they paid. Never the embedding.

## Dependencies

Go standard library only: `math`, `crypto/*`, `encoding/*`, `net`, `sync`. No third-party imports. Every dependency is attack surface.

## Package Structure

```
auction/         # Vendored auction math (independently auditable)
  types.go       #   CoreBid, ScoredBid, AuctionResult
  embedding.go   #   ComputeEmbeddingScore, SquaredEuclideanDistance
  vcg.go         #   ComputeVCGPayment
  auction.go     #   RunAuction (orchestrator)
  floor.go       #   EnforceBidFloor
  ranking.go     #   RankScoredBids (crypto-random tie-breaking)
types.go         # Message types (AuctionRequest, AuctionResponse, snapshots)
crypto.go        # RSA-OAEP + AES-256-GCM hybrid encryption/decryption
keymanager.go    # RSA-2048 keypair generation and PEM export
state.go         # Thread-safe stores for positions and budgets
process.go       # ProcessPrivateAuction (decrypt → filter → auction → payment)
listen.go        # Vsock listener with TCP fallback
```

## Verify

```bash
# Confirm all imports are stdlib
go list -f '{{join .Imports "\n"}}' ./...

# Build and test
go test -count=1 ./...
```

## Drift Risk

The `auction/` subdirectory is a vendored copy of the auction math from the [adserver repo](https://github.com/kimjune01/vectorspace-adserver). The adserver has a cross-validation test (`auction_crosscheck_test.go`) that runs identical inputs through both copies and asserts bit-identical results. Any divergence fails CI.

## Protocol

The exchange parent communicates via JSON envelope (`{type, payload}`) over vsock (production) or TCP (development):

| Message | Direction | Purpose |
|---------|-----------|---------|
| `ping` / `pong` | ↔ | Health check |
| `key_request` / `key_response` | ← → | Attestation + public key |
| `sync_positions` | → | Push advertiser positions |
| `sync_budgets` | → | Push budget snapshots |
| `auction_request` / `auction_response` | → ← | Run an auction |

## What Enters and Exits

**In:** Encrypted embedding, τ threshold, advertiser positions, budget snapshots.

**Out:** Winner ID, VCG payment, currency, bid count.

**Never exits:** The query embedding, individual scores, which advertisers were filtered.
