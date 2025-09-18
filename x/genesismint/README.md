# GenesisMint Module (Consumer Side)

The GenesisMint module mints the consumer chain’s native token from trusted
provider state at genesis and confirms the claim on the provider via ICA.
It is designed to run once (idempotent), then disable itself automatically.

## What It Does

- At genesis (InitGenesis):
  - Validates and applies each MintIntent by verifying a Merkle proof of a
    key/value from the provider (ICS‑23 membership against a trusted root).
  - Mints `mint_denom` to the specified `recipient` and marks the escrow as
    claimed locally.
  - Enqueues a “pending claim” to notify the provider chain via Interchain
    Accounts (ICA) that this escrow was claimed on the consumer.

- After genesis (BeginBlocker):
  - Ensures the designated ICA (owner/connection) is registered and active
    (debounced; safe to call every block).
  - Flushes pending claims by sending a provider message via ICA.
  - Tracks in‑flight packets and removes pending items only after success acks.
  - Retries on ack errors and timeouts; watchdog resubmits if no ack/timeout is
    observed within the configured timeout window.
  - When no pending nor in‑flight items remain, sets a “done” flag to stop any
    further BeginBlock work.

## Security Model

- Proof verification (genesis only):
  - Uses ICS‑23 membership proofs against either:
    - the provider client’s consensus state root (if present), or
    - an optional trusted genesis root from params (for bootstrap flows).
  - Enforces provider chain id and allowed denom checks before minting.

- ICA claim confirmation:
  - Claims are sent to the provider via ICA (ICS‑27). Pending entries are
    cleared only after a success acknowledgement from the provider.
  - Timeout and ack‑error cause retry. A watchdog will resubmit if no ack nor
    timeout arrives within the configured duration (e.g., relayer stalled).

## Runtime Flow (Overview)

1) InitGenesis
   - Validate GenesisState.
   - For each MintIntent: verify proof → mint → mark claimed → enqueue pending.

2) BeginBlock (until done)
   - Ensure ICA registered (idempotent).
   - If channel open and ICA address is known, flush up to N pending claims:
     - Submit provider message; record (channel, sequence) → (provider, escrow)
       mapping and mark the claim as in‑flight (prevents duplicate sends).
   - On ICA ack success (via middleware): clear in‑flight and pending.
   - On ack error/timeout: clear in‑flight only (keeps pending for retry).
   - Watchdog: if a claim stays in‑flight for ≥ timeout seconds without ack or
     timeout, clear in‑flight to allow retry.
   - If pending=0 and in‑flight=0: set “done” and stop.

## Configuration (via genesis params)

The module accepts extra ICA config fields in `app_state.genesismint.params`.
Both snake_case and camelCase are supported.

- ica_controller_connection_id (string)  |  icaControllerConnectionId
- ica_owner (string)                     |  icaOwner
- ica_tx_timeout_seconds (uint)          |  icaTxTimeoutSeconds
- ica_max_claims_per_block (uint)        |  icaMaxClaimsPerBlock | icaMaxClaimPerBlock

Defaults
- connection: `connection-0`
- owner: `mintburn-claims`
- timeout seconds: `300`
- max claims per block: `10`

Example
```json
{
  "genesismint": {
    "params": {
      "provider_client_id": "07-tendermint-0",
      "provider_chain_id": "maany-local-1",
      "allowed_provider_denom": "umaany",
      "mint_denom": "umaany",
      "use_genesis_trusted_root": true,
      "genesis_trusted_root": {
        "revision_number": 0,
        "revision_height": 3,
        "hash": "...base64 or bytes..."
      },
      "icaControllerConnectionId": "connection-0",
      "icaOwner": "mintburn-claims",
      "icaTxTimeoutSeconds": 180,
      "icaMaxClaimsPerBlock": 10
    },
    "mints": [ /* MintIntent list */ ]
  }
}
```

## Integration Notes

- App wiring
  - Keeper requires: Bank, IBC ClientKeeper, ICA Controller Keeper, ICA MsgServer,
    and IBC ConnectionKeeper (to build ICA metadata with host connection id).
  - The ICA controller IBC route is wrapped with a small middleware to capture
    acks/timeouts and trigger pending removal or retry.

- Relayer config
  - Hermes must relay ICS‑27 (ICA) handshakes and EXECUTE_TX packets. Ensure
    packet filter allows ports `icacontroller-<owner>` and `icahost` (or allow all).

- Timeouts
  - Set `ica_tx_timeout_seconds` to a sufficiently large value (e.g., 120–300) to
    tolerate relayer latency, block time skew, and network hiccups.

## Troubleshooting

- BeginBlocker not running: ensure the module implements appmodule.HasBeginBlocker
  and is in ModuleManager’s begin‑block order; also ensure blocks are produced.
- Repeated ICA registration attempts: the keeper debounces; if you still see
  repeated channel inits, Hermes may be filtering ICA packets.
- Ack errors/timeouts: the module automatically retries; raise timeout seconds
  if the relayer is slow or clocks are skewed.
- Stuck “flushing pending claims”: fixed by in‑flight guard + watchdog; check
  logs for ack/timeout events and verify Hermes is relaying.

## Logs (selected)

- InitGenesis: start/end; each processed mint intent
- Proof: using consensus root vs genesis trusted root; ICS‑23 verified
- Mint/send: minted coins, sent to recipient, enqueued pending
- ICA: checking registration; channel active; sent claim (awaiting ack);
  mapped packet; ack success/error; timeout; watchdog retry; done

## Internals (Pointers)

- Keeper core: `x/genesismint/keeper/keeper.go`
  - ProcessGenesisMint: proof verify + mint + queue
  - BeginBlocker: ICA ensure + flush + retry orchestration
  - Ack/timeout handlers: pending removal / inflight clearing
- Types/keys: `x/genesismint/types/keys.go`
  - Claimed index, Pending queue, In‑flight, Packet map, Done, Config
- Module wiring: `x/genesismint/module/module.go`, `x/genesismint/module/ica_middleware.go`

## Extending

- Switch to interchaintxs MsgSubmitTx if you want tx IDs and richer callbacks.
- Add queries to expose pending/in‑flight state and ICA config.
- Tune retry policy (exponential backoff, max attempts) as needed.

