package keeper

import (
    "encoding/base64"
    "encoding/hex"
    "fmt"
    "strconv"
    "time"

    sdkmath "cosmossdk.io/math"

    storetypes "cosmossdk.io/store/types"
    "github.com/cosmos/cosmos-sdk/codec"
    codectypes "github.com/cosmos/cosmos-sdk/codec/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
    icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
    icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
    clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
    commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"

    "github.com/maany-xyz/maany-dex/v5/x/genesismint/types"
    // provider message packed as Any (no direct provider module import)
)
type hasRoot interface{ GetRoot() *commitmenttypes.MerkleRoot }

// --- Keeper ---

type Keeper struct {
    cdc          codec.BinaryCodec
    storeKey     storetypes.StoreKey
    bank         BankKeeper
    clientKeeper ClientKeeper
    icaCtrlKeeper ICAControllerKeeper
    icaMsgServer  ICAControllerMsgServer
    connKeeper   ConnectionKeeper
}

func NewKeeper(
    cdc codec.BinaryCodec,
    key storetypes.StoreKey,
    bk BankKeeper,
    ck ClientKeeper,
    icaCtrl ICAControllerKeeper,
    icaMsg ICAControllerMsgServer,
    connKeeper ConnectionKeeper,
) Keeper {
    return Keeper{
        cdc:          cdc,
        storeKey:     key,
        bank:         bk,
        clientKeeper: ck,
        icaCtrlKeeper: icaCtrl,
        icaMsgServer:  icaMsg,
        connKeeper:   connKeeper,
    }
}

// Exported setters for InitGenesis config wiring
func (k Keeper) SetICAConnectionID(ctx sdk.Context, v string) { k.setICAConnectionID(ctx, v) }
func (k Keeper) SetICAOwner(ctx sdk.Context, v string)        { k.setICAOwner(ctx, v) }
func (k Keeper) SetICATimeoutSeconds(ctx sdk.Context, v uint64) { k.setICATimeoutSeconds(ctx, v) }
func (k Keeper) SetMaxClaimsPerBlock(ctx sdk.Context, v int)  { k.setMaxClaimsPerBlock(ctx, v) }

// --- Claimed index ---

// claimed key: claimed|<providerChainID>|<escrowID>
func (k Keeper) setClaimed(ctx sdk.Context, providerChainID, escrowID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ClaimedKey(providerChainID, escrowID), []byte{1})
}

func (k Keeper) isClaimed(ctx sdk.Context, providerChainID, escrowID string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.ClaimedKey(providerChainID, escrowID))
}

// --- Core flow: verify (optional) + mint + mark claimed ---

// ProcessGenesisMint executes one MintIntent from genesis.
// put this near the top of keeper.go (outside the func) if you haven't already:
// ProcessGenesisMint executes one MintIntent from genesis.
func (k Keeper) ProcessGenesisMint(
    ctx sdk.Context,
    params *types.Params,
    intent *types.MintIntent,
    strictProof bool,
) error {
    ctx.Logger().Info(
        "genesismint: processing mint intent",
        "escrow_id", intent.GetEscrowId(),
        "recipient", intent.GetRecipient(),
        "amount", intent.GetAmountValue(),
        "denom", params.GetMintDenom(),
        "strict_proof", strictProof,
    )
	// Idempotency
	if k.isClaimed(ctx, intent.ProviderChainId, intent.EscrowId) {
		return nil
	}

	// Sanity
	if params == nil {
		return fmt.Errorf("params missing")
	}
	if err := params.ValidateBasic(); err != nil {
		return fmt.Errorf("params invalid: %w", err)
	}
	if intent == nil {
		return fmt.Errorf("mint intent nil")
	}
	if intent.ProviderChainId != params.ProviderChainId {
		return fmt.Errorf("provider_chain_id mismatch: %s != %s", intent.ProviderChainId, params.ProviderChainId)
	}
	if intent.AmountDenom != params.AllowedProviderDenom {
		return fmt.Errorf("amount_denom not allowed: %s", intent.AmountDenom)
	}
	if len(intent.KeyPath) != 2 {
		return fmt.Errorf("key_path must be [storeName, hexKey], got %v", intent.KeyPath)
	}
	storeName := intent.KeyPath[0]
	keyBytes, err := hex.DecodeString(intent.KeyPath[1])
	if err != nil {
		return fmt.Errorf("bad hex key: %w", err)
	}
	valBytes, err := base64.StdEncoding.DecodeString(intent.Value)
	if err != nil {
		return fmt.Errorf("bad base64 value: %w", err)
	}

	// STRICT PROOF (prod): verify against IBC client OR (genesis) trusted root.
    if strictProof {
        h := clienttypes.NewHeight(intent.ProofHeightRevisionNumber, intent.ProofHeightRevisionHeight)

        consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, params.ProviderClientId, h)

        var root *commitmenttypes.MerkleRoot

        if found {
            // assert Tendermint consensus state and use its root
            tmCS, ok := consensusState.(hasRoot)
            if !ok {
                return fmt.Errorf("expected tendermint consensus state, got %T", consensusState)
            }
            root = tmCS.GetRoot()
            if root == nil {
                return fmt.Errorf("nil commitment root at %s", h.String())
            }
            ctx.Logger().Info("genesismint: using consensus state root for proof", "height", h.String())
        } else {
            // 🔁 GENESIS FALLBACK: verify against a trusted root from params
            // (You must add these fields to Params in your proto/types)
            ctx.Logger().Info("genesismint: consensus state missing, trying genesis trusted root",
                "client_id", params.ProviderClientId,
                "rev", intent.ProofHeightRevisionNumber,
                "height", intent.ProofHeightRevisionHeight,
            )
            if params.UseGenesisTrustedRoot &&
                params.GenesisTrustedRoot != nil &&
                params.GenesisTrustedRoot.RevisionNumber == intent.ProofHeightRevisionNumber &&
                params.GenesisTrustedRoot.RevisionHeight == intent.ProofHeightRevisionHeight &&
                len(params.GenesisTrustedRoot.Hash) > 0 {

                root = &commitmenttypes.MerkleRoot{Hash: params.GenesisTrustedRoot.Hash}
            } else {
                return fmt.Errorf(
                    "no consensus state for client %s at height %d-%d and no usable genesis_trusted_root",
                    params.ProviderClientId, intent.ProofHeightRevisionNumber, intent.ProofHeightRevisionHeight,
                )
            }
        }

        // ICS-23 verify
        specs := commitmenttypes.GetSDKSpecs()
        // Path must be [storeName, key] for SDK multi-store -> IAVL chained proofs.
        // First segment selects the sub-store, second is the raw KV key bytes.
        path := commitmenttypes.NewMerklePath(storeName, string(keyBytes))
        if err := intent.MerkleProof.VerifyMembership(specs, root, path, valBytes); err != nil {
            return fmt.Errorf("ics23 verify membership failed: %w", err)
        }
        ctx.Logger().Info("genesismint: ICS-23 membership verified")
    }

	// Mint + send (v0.50 bank uses context.Context)
	amt, ok := sdkmath.NewIntFromString(intent.AmountValue)
	if !ok {
		return fmt.Errorf("bad amount_value: %s", intent.AmountValue)
	}
	coins := sdk.NewCoins(sdk.NewCoin(params.MintDenom, amt))

    // In SDK v0.50+, sdk.Context implements context.Context; pass ctx directly.
    if err := k.bank.MintCoins(ctx, types.ModuleName, coins); err != nil {
        return fmt.Errorf("mint coins: %w", err)
    }
    ctx.Logger().Info("genesismint: minted coins", "amount", coins.String())
    rcpt, err := sdk.AccAddressFromBech32(intent.Recipient)
    if err != nil {
        return fmt.Errorf("recipient bech32: %w", err)
    }
    if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, rcpt, coins); err != nil {
        return fmt.Errorf("send to recipient: %w", err)
    }
    ctx.Logger().Info("genesismint: sent coins to recipient", "recipient", intent.Recipient, "amount", coins.String())

    // Mark claimed
    k.setClaimed(ctx, intent.ProviderChainId, intent.EscrowId)
    // Enqueue a pending claim to notify provider via ICA later.
    k.enqueuePendingClaim(ctx, intent.ProviderChainId, intent.EscrowId)
    ctx.Logger().Info("genesismint: marked claimed and enqueued pending ICA claim",
        "provider_chain_id", intent.ProviderChainId,
        "escrow_id", intent.EscrowId,
    )

	// Event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("genesismint_minted",
			sdk.NewAttribute("escrow_id", intent.EscrowId),
			sdk.NewAttribute("recipient", intent.Recipient),
			sdk.NewAttribute("amount", coins.String()),
			sdk.NewAttribute("proof_height", fmt.Sprintf("%d", intent.ProofHeightRevisionHeight)),
		),
	})
	return nil
}

// ---- Pending claim queue helpers ----

func (k Keeper) enqueuePendingClaim(ctx sdk.Context, providerChainID, escrowID string) {
    store := ctx.KVStore(k.storeKey)
    store.Set(types.PendingClaimKey(providerChainID, escrowID), []byte{1})
}

func (k Keeper) deletePendingClaim(ctx sdk.Context, providerChainID, escrowID string) {
    store := ctx.KVStore(k.storeKey)
    store.Delete(types.PendingClaimKey(providerChainID, escrowID))
}

// Iterate a limited number of pending claims, returning pairs (providerChainID, escrowID).
func (k Keeper) listPendingClaims(ctx sdk.Context, limit int) [][2]string {
    store := ctx.KVStore(k.storeKey)
    it := storetypes.KVStorePrefixIterator(store, types.PendingClaimPrefix)
    defer it.Close()
    out := make([][2]string, 0, limit)
    for it.Valid() && len(out) < limit {
        // key format: PendingClaimPrefix + "pclaim|" + provider + "|" + escrow
        key := it.Key()
        // strip prefix byte + "pclaim|"
        s := string(key[len(types.PendingClaimPrefix):])
        // s == "pclaim|<provider>|<escrow>"
        // find provider and escrow by splitting on '|'
        // avoid allocations: simple parse
        const head = "pclaim|"
        if len(s) > len(head) && s[:len(head)] == head {
            rest := s[len(head):]
            // find next '|'
            for i := 0; i < len(rest); i++ {
                if rest[i] == '|' {
                    prov := rest[:i]
                    esc := rest[i+1:]
                    out = append(out, [2]string{prov, esc})
                    break
                }
            }
        }
        it.Next()
    }
    return out
}

// hasAnyPendingClaims returns true if there is at least one pending claim queued.
func (k Keeper) hasAnyPendingClaims(ctx sdk.Context) bool {
    store := ctx.KVStore(k.storeKey)
    it := storetypes.KVStorePrefixIterator(store, types.PendingClaimPrefix)
    defer it.Close()
    return it.Valid()
}

// ---- ICA submission on BeginBlock ----

const (
    defaultICAConnectionID   = "connection-0"
    defaultICAOwnerID        = "mintburn-claims"
    defaultICATimeoutSeconds = 300
    defaultMaxClaimsPerBlock = 10
)

// ---- Config getters/setters ----
func (k Keeper) getICAConnectionID(ctx sdk.Context) string {
    bz := ctx.KVStore(k.storeKey).Get(types.ConfigICAConnectionIDKey())
    if len(bz) == 0 { return defaultICAConnectionID }
    return string(bz)
}
func (k Keeper) setICAConnectionID(ctx sdk.Context, v string) { ctx.KVStore(k.storeKey).Set(types.ConfigICAConnectionIDKey(), []byte(v)) }

func (k Keeper) getICAOwner(ctx sdk.Context) string {
    bz := ctx.KVStore(k.storeKey).Get(types.ConfigICAOwnerKey())
    if len(bz) == 0 { return defaultICAOwnerID }
    return string(bz)
}
func (k Keeper) setICAOwner(ctx sdk.Context, v string) { ctx.KVStore(k.storeKey).Set(types.ConfigICAOwnerKey(), []byte(v)) }

func (k Keeper) getICATimeoutSeconds(ctx sdk.Context) uint64 {
    bz := ctx.KVStore(k.storeKey).Get(types.ConfigICATimeoutSecondsKey())
    if len(bz) == 0 { return defaultICATimeoutSeconds }
    u, err := strconv.ParseUint(string(bz), 10, 64)
    if err != nil { return defaultICATimeoutSeconds }
    return u
}
func (k Keeper) setICATimeoutSeconds(ctx sdk.Context, v uint64) { ctx.KVStore(k.storeKey).Set(types.ConfigICATimeoutSecondsKey(), []byte(strconv.FormatUint(v,10))) }

func (k Keeper) getMaxClaimsPerBlock(ctx sdk.Context) int {
    bz := ctx.KVStore(k.storeKey).Get(types.ConfigMaxClaimsPerBlockKey())
    if len(bz) == 0 { return defaultMaxClaimsPerBlock }
    u, err := strconv.ParseUint(string(bz), 10, 32)
    if err != nil { return defaultMaxClaimsPerBlock }
    return int(u)
}
func (k Keeper) setMaxClaimsPerBlock(ctx sdk.Context, v int) { ctx.KVStore(k.storeKey).Set(types.ConfigMaxClaimsPerBlockKey(), []byte(strconv.FormatUint(uint64(v),10))) }

// BeginBlocker attempts to ensure ICA registration and flush a few pending claims to provider.
func (k Keeper) BeginBlocker(ctx sdk.Context) {
    // If module finished its one-shot workflow, skip all further work.
    if k.isDone(ctx) {
        return
    }
    ctx.Logger().Info("genesismint: begin block tick", "height", ctx.BlockHeight())
    owner := k.getICAOwner(ctx)
    conn := k.getICAConnectionID(ctx)

    // Ensure registration (idempotent)
    k.ensureICARegistered(ctx, conn, owner)

    // If channel not active, skip flushing
    portID, err := icatypes.NewControllerPortID(owner)
    if err != nil {
        ctx.Logger().Error("genesismint: bad ICA owner for portID", "owner", owner, "err", err)
        return
    }
    if _, found := k.icaCtrlKeeper.GetActiveChannelID(ctx, conn, portID); !found {
        ctx.Logger().Info("genesismint: ICA channel not active yet; skipping claim flush",
            "connection", conn, "owner", owner, "port_id", portID,
        )
        return
    }

    // Obtain ICA address on provider to set as sender
    icaAddr, ok := k.icaCtrlKeeper.GetInterchainAccountAddress(ctx, conn, portID)
    if !ok || icaAddr == "" {
        // not ready yet
        ctx.Logger().Info("genesismint: ICA address not available yet; skipping claim flush",
            "connection", conn, "owner", owner, "port_id", portID,
        )
        return
    }

    // Flush a bounded number of pending claims
    pending := k.listPendingClaims(ctx, k.getMaxClaimsPerBlock(ctx))
    if len(pending) == 0 {
        // No more claims to flush and ICA is active + address available: mark done to disable future checks.
        k.setDone(ctx)
        ctx.Logger().Info("genesismint: no pending claims; marking module done and disabling further BeginBlock work")
        return
    }
    ctx.Logger().Info("genesismint: flushing pending claims via ICA",
        "count", len(pending), "connection", conn, "port_id", portID,
    )
    for _, pair := range pending {
        providerChainID, escrowID := pair[0], pair[1]

        // Build provider message Any without importing provider module
        anyMsg, err := buildProviderMarkClaimAny(icaAddr, escrowID, ctx.ChainID())
        if err != nil {
            ctx.Logger().Error("genesismint: build provider msg failed", "err", err, "escrow_id", escrowID)
            continue
        }

        // Serialize CosmosTx with single message
        cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
        bz, err := k.cdc.Marshal(cosmosTx)
        if err != nil {
            ctx.Logger().Error("genesismint: marshal CosmosTx failed", "err", err)
            continue
        }

        packet := icatypes.InterchainAccountPacketData{
            Type: icatypes.EXECUTE_TX,
            Data: bz,
            Memo: "",
        }

        // SendTx via ICA controller
        _, err = k.icaMsgServer.SendTx(ctx, &icacontrollertypes.MsgSendTx{
            Owner:           owner,
            ConnectionId:    conn,
            PacketData:      packet,
            RelativeTimeout: uint64(time.Duration(k.getICATimeoutSeconds(ctx)) * time.Second),
        })
        if err != nil {
            ctx.Logger().Error("genesismint: ICA SendTx failed", "err", err, "escrow_id", escrowID)
            // keep for retry
            continue
        }

        // For simplicity, remove immediately; provider is idempotent. A more robust
        // design would remove on ack success.
        k.deletePendingClaim(ctx, providerChainID, escrowID)
        ctx.Logger().Info("genesismint: sent provider claim via ICA",
            "escrow_id", escrowID,
            "provider_chain_id", providerChainID,
            "sender_ica", icaAddr,
        )
    }
}

// ---- Minimal protobuf encoding for /maany.mintburn.v1.MsgMarkEscrowClaimed ----
// message MsgMarkEscrowClaimed { string sender = 1; string escrow_id = 2; string consumer_chain_id = 3; }
// We avoid importing provider stubs by building the Any payload manually.

const providerMsgMarkEscrowClaimedTypeURL = "/maany.mintburn.v1.MsgMarkEscrowClaimed"

func buildProviderMarkClaimAny(sender, escrowID, consumerChainID string) (*codectypes.Any, error) {
    // Helper to encode a single string field as protobuf: key (varint), len (varint), bytes
    // key = (field_number << 3) | 2 (wire type 2 = length-delimited)
    encField := func(fieldNum int, s string, dst *[]byte) {
        key := uint64((fieldNum << 3) | 2)
        *dst = appendVarint(*dst, key)
        *dst = appendVarint(*dst, uint64(len(s)))
        *dst = append(*dst, []byte(s)...)
    }
    var b []byte
    encField(1, sender, &b)
    encField(2, escrowID, &b)
    encField(3, consumerChainID, &b)
    return &codectypes.Any{TypeUrl: providerMsgMarkEscrowClaimedTypeURL, Value: b}, nil
}

func appendVarint(dst []byte, x uint64) []byte {
    for x >= 0x80 {
        dst = append(dst, byte(x)|0x80)
        x >>= 7
    }
    return append(dst, byte(x))
}

// ---- ICA registration pending flag helpers ----
func (k Keeper) setICAPending(ctx sdk.Context, connectionID, owner string) {
    store := ctx.KVStore(k.storeKey)
    store.Set(types.ICAPendingKey(connectionID, owner), []byte{1})
}

func (k Keeper) clearICAPending(ctx sdk.Context, connectionID, owner string) {
    store := ctx.KVStore(k.storeKey)
    store.Delete(types.ICAPendingKey(connectionID, owner))
}

func (k Keeper) isICAPending(ctx sdk.Context, connectionID, owner string) bool {
    store := ctx.KVStore(k.storeKey)
    return store.Has(types.ICAPendingKey(connectionID, owner))
}

// ensureICARegistered registers the interchain account for the given (connection, owner)
// if an active channel does not already exist. It is safe to call repeatedly.
func (k Keeper) ensureICARegistered(ctx sdk.Context, connectionID, owner string) {
    params := k.icaCtrlKeeper.GetParams(ctx)
    ctx.Logger().Info("genesismint: checking ICA registration",
        "connection", connectionID,
        "owner", owner,
        "height", ctx.BlockHeight(),
        "ica_controller_enabled", params.ControllerEnabled,
    )
    portID, err := icatypes.NewControllerPortID(owner)
    if err != nil {
        ctx.Logger().Error("genesismint: bad ICA owner for portID", "owner", owner, "err", err)
        return
    }
    if _, found := k.icaCtrlKeeper.GetActiveChannelID(ctx, connectionID, portID); found {
        if k.isICAPending(ctx, connectionID, owner) {
            k.clearICAPending(ctx, connectionID, owner)
            ctx.Logger().Info("genesismint: ICA became active; cleared pending flag", "connection", connectionID, "owner", owner, "port_id", portID)
        } else {
            ctx.Logger().Info("genesismint: ICA channel already active", "connection", connectionID, "owner", owner, "port_id", portID)
        }
        return
    }

    // If we already attempted registration, just wait for relayer to complete handshake.
    if k.isICAPending(ctx, connectionID, owner) {
        ctx.Logger().Info("genesismint: ICA registration already pending; waiting for channel to open",
            "connection", connectionID, "owner", owner, "port_id", portID)
        return
    }

    // Trigger registration once, then mark pending to avoid repeated channel inits per block.
    // Build explicit ICA metadata including both controller and host connection IDs
    var version string
    if connEnd, ok := k.connKeeper.GetConnection(ctx, connectionID); ok {
        cp := connEnd.GetCounterparty()
        hostConnID := cp.GetConnectionID()
        if hostConnID == "" {
            // Fallback for setups where counterparty connection ID is not populated in state; assume symmetric ID.
            hostConnID = connectionID
            ctx.Logger().Info("genesismint: counterparty connection id empty; using controller id as host id", "connection", connectionID)
        }
        md := icatypes.Metadata{
            Version:                icatypes.Version,
            ControllerConnectionId: connectionID,
            HostConnectionId:       hostConnID,
            Address:                "",
            Encoding:               icatypes.EncodingProtobuf,
            TxType:                 icatypes.TxTypeSDKMultiMsg,
        }
        bz := icatypes.ModuleCdc.MustMarshalJSON(&md)
        version = string(bz)
        ctx.Logger().Info("genesismint: built ICA metadata", "controller_conn", connectionID, "host_conn", hostConnID, "version", version)
    } else {
        // fallback: let module construct version; may fail if host expects explicit host_conn
        version = ""
        ctx.Logger().Info("genesismint: connection not found when building metadata; falling back to default version", "connection", connectionID)
    }

    _, err = k.icaMsgServer.RegisterInterchainAccount(ctx, &icacontrollertypes.MsgRegisterInterchainAccount{
        Owner:        owner,
        ConnectionId: connectionID,
        Version:      version,
    })
    if err != nil {
        ctx.Logger().Error("genesismint: ICA registration attempt failed", "err", err, "connection", connectionID, "owner", owner)
    } else {
        k.setICAPending(ctx, connectionID, owner)
        ctx.Logger().Info("genesismint: initiated ICA registration", "connection", connectionID, "owner", owner, "port_id", portID)
    }
}

// ---- Done flag helpers ----
func (k Keeper) setDone(ctx sdk.Context) {
    store := ctx.KVStore(k.storeKey)
    store.Set(types.DoneKey(), []byte{1})
}

func (k Keeper) isDone(ctx sdk.Context) bool {
    store := ctx.KVStore(k.storeKey)
    return store.Has(types.DoneKey())
}
