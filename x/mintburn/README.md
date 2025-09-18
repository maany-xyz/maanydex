# MintBurn Module (Consumer Side)

This module mints the consumer chain’s native (DEX) token upon receiving an ICS‑20 transfer from the registered CCV provider. It uses the CCV Consumer keeper as the source of truth to bind the privileged minting path to the provider’s authenticated IBC connection and client.

## What It Does

- Privileged mint on Provider → Consumer (DEX) transfers:
  - Verifies the incoming ICS‑20 packet arrived over a transfer channel bound to the CCV provider client.
  - Checks the base denom is allow‑listed.
  - Mints `DexNativeDenom` to the packet receiver and prevents ICS‑20 voucher minting for that packet.
- Accounting and safety:
  - Replay protection for packets (per‑packet proof ID).
  - Tracks “mirror supply” minted on Consumer.
- Optional DEX → Provider flow:
  - On successful ACK, burns escrowed native tokens and decrements mirror supply (see code note).

## Security Model

- CCV binding (soft‑fail):
  - On channel handshake (transfer), the module looks up the local transfer channel’s connection and requires its local client ID to equal the CCV Consumer keeper’s provider client ID.
  - If it matches, the module allow‑lists the local channel for privileged minting. Otherwise, it does not allow‑list (channel still opens and can be used for normal ICS‑20 vouchers).
- On packet receive, the same binding is enforced again before minting. Unauthorized paths are rejected for minting and not forwarded as privileged packets.
- DestinationChannel is used on receive (local channel ID); SourceChannel is the remote.

This prevents malicious chains from spoofing chain‑id/denom: they cannot produce valid headers for the CCV provider client tracked on the consumer.

## Code Map

- `x/mintburn/module/middleware.go`
  - IBC transfer middleware: channel lifecycle and packet handlers
  - Channel allow‑listing (soft‑fail) bound to CCV provider client
  - OnRecvPacket privileged‑path checks, denom validation, minting
- `x/mintburn/keeper/keeper.go`
  - Module storage, params, mint/burn helpers, replay guard, mirror supply
- `x/mintburn/types/params.go`
  - Params definition and validation

## Params

- `authority` (string): optional admin address (not used by default)
- `pause` (bool): if true, rejects privileged minting
- `allowed_base_denoms` ([]string): base denoms accepted from the provider (e.g., `["stake"]` or `["umaany"]`)
- `provider_chain_ids` ([]string): legacy/aux parameter; CCV binding is the primary security
- `dex_native_denom` (string): local denom to mint on privileged path (e.g., `"umaany"`)

Defaults are defined in `types/params.go`. Params are persisted in module KV via `keeper.SetParams` at genesis.

## Genesis Example

```json
{
  "mintburn": {
    "params": {
      "pause": false,
      "allowed_base_denoms": ["umaany"],
      "provider_chain_ids": ["maany-local-1"],
      "dex_native_denom": "umaany"
    }
  }
}
```

Note: CCV binding uses the Consumer keeper’s provider client ID at runtime and does not require additional mintburn params.

## Channel Allow‑Listing (Soft‑Fail)

- On `OnChanOpenAck/OnChanOpenConfirm` for `port=transfer`:
  - Resolve the local connection from the channel.
  - `localClientID := connection.ClientId`
  - `providerClientID := ConsumerKeeper.GetProviderClientID(ctx)`
  - If equal, store the local channel ID in the allow‑list; else, skip (channel still opens, but not privileged).

## Receive Path Checks

For each ICS‑20 packet on the consumer:

1) `DestinationPort == "transfer"`
2) `DestinationChannel` is allow‑listed (local channel ID)
3) Local connection’s client ID equals CCV provider client ID
4) `params.pause == false`
5) Base denom (after trace stripping) is in `params.allowed_base_denoms`
6) Packet not replayed (unique proof key)

If all pass, mint `params.dex_native_denom` to `data.receiver` and ACK success. Otherwise, either forward to ICS‑20 voucher path or return error ACK (see logs below).

## Logging Guide

- Channel handshake
  - `mintburn: set allowed channel (via CCV client bind)` — privileged channel recorded
  - `mintburn: channel client mismatch; not allowlisting` — not privileged
- On receive (non‑privileged reasons)
  - `non-transfer destination port; forwarding`
  - `channel not allowlisted; forwarding`
  - `unauthorized transfer path; rejecting` (client mismatch)
  - `base denom not allowed; forwarding`
  - `invalid receiver address` / `invalid amount`
- On privileged mint
  - `mintburn: minted mirrored on DEX` (amount, receiver, proof)

## Integration Notes

- App wiring must pass the CCV Consumer keeper into the mintburn keeper:
  - `NewKeeper(..., ChannelKeeper, ConnectionKeeper, ClientKeeper, &ConsumerKeeper)`
- The middleware wraps the IBC transfer module; other chains can still open transfer channels (soft‑fail); only CCV‑bound channels are privileged.

## Troubleshooting

- Packet forwards to vouchers instead of mint:
  - Channel not allow‑listed (check handshake logs)
  - Client mismatch against CCV provider client (check `local_client_id` vs `provider_client_id`)
  - Base denom not in `allowed_base_denoms`
  - Module paused
- Duplicate packet: see `duplicate proof` error

## Extending

- To hard‑fail unauthorized handshakes, return an error in channel open handlers (will block non‑provider ICS‑20 channels). Keep soft‑fail for general interoperability.
- Consider a query to expose the current allow‑listed channels and params for monitoring.

