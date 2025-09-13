package keeper

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"

	"github.com/maany-xyz/maany-dex/v5/x/genesismint/types"
)
type hasRoot interface{ GetRoot() *commitmenttypes.MerkleRoot }

// --- Keeper ---

type Keeper struct {
	cdc          codec.BinaryCodec
	storeKey     storetypes.StoreKey
	bank         BankKeeper
	clientKeeper ClientKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	bk BankKeeper,
	ck ClientKeeper,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeKey:     key,
		bank:         bk,
		clientKeeper: ck,
	}
}

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
func (k Keeper) ProcessGenesisMint(
	ctx sdk.Context,
	params *types.Params,
	intent *types.MintIntent,
	strictProof bool, // set true to verify ICS-23, false to bypass (dev only)
) error {
	// Idempotency
	if k.isClaimed(ctx, intent.ProviderChainId, intent.EscrowId) {
		return nil
	}

	// Params / intent sanity
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

	// OPTIONAL: decode provider Escrow protobuf to assert fields
	// If you import the provider's mintburn types:
	//   providertypes "github.com/maany-xyz/maany-provider/x/mintburn/types"
	// var escrow providertypes.Escrow
	// if err := k.cdc.Unmarshal(valBytes, &escrow); err != nil {
	//     return fmt.Errorf("unmarshal provider escrow: %w", err)
	// }
	// if escrow.GetAmount().Denom != intent.AmountDenom || escrow.GetAmount().Amount != intent.AmountValue {
	//     return fmt.Errorf("escrow amount mismatch vs intent")
	// }
	// if escrow.GetConsumerChainId() != ctx.ChainID() {
	//     return fmt.Errorf("escrow consumer_chain_id != this chain")
	// }

	// Strict ICS-23 proof verification (recommended for prod)
	if strictProof {
		h := clienttypes.NewHeight(intent.ProofHeightRevisionNumber, intent.ProofHeightRevisionHeight)

		consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, params.ProviderClientId, h)
		if !found {
			return fmt.Errorf("no consensus state for client %s at height %s",
				params.ProviderClientId, h.String())
		}

		// Assert Tendermint consensus state to access Root
		tmCS, ok := consensusState.(hasRoot)
		if !ok {
			return fmt.Errorf("expected tendermint consensus state, got %T", consensusState)
		}

		root := tmCS.GetRoot()
		if root == nil {
			return fmt.Errorf("nil commitment root at %s", h.String())
		}


		
		// 2) ICS-23 proof specs for Cosmos SDK stores (IAVL/Tendermint)
		specs := commitmenttypes.GetSDKSpecs() // []*ics23.ProofSpec

		// 3) Build the Merkle path **including the key**
		// For ABCI KV proofs, v8 expects the key to be encoded into the path.
		// Use lowercase hex and NO "0x" prefix.
		hexKey := strings.ToLower(hex.EncodeToString(keyBytes))

		// Preferred form that works with ABCI: ["store", <storeName>, "key", <hexKey>]
		path := commitmenttypes.NewMerklePath("store", storeName, "key", hexKey)

		if err := intent.MerkleProof.VerifyMembership(specs, root, path, valBytes); err != nil {
			// If this fails with a path mismatch, try the 3-segment fallback:
			//   path = commitmenttypes.NewMerklePath("store", storeName, "key")
			//   err = intent.MerkleProof.VerifyMembership(specs, root, path, valBytes)
			// and ensure hexKey casing matches what your proof util emits.
			return fmt.Errorf("ics23 verify membership failed: %w", err)
		}
	}

	// Mint + send
	amt, ok := sdkmath.NewIntFromString(intent.AmountValue)
	if !ok {
		return fmt.Errorf("bad amount_value: %s", intent.AmountValue)
	}
	coins := sdk.NewCoins(sdk.NewCoin(params.MintDenom, amt))
	if err := k.bank.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return fmt.Errorf("mint coins: %w", err)
	}
	rcpt, err := sdk.AccAddressFromBech32(intent.Recipient)
	if err != nil {
		return fmt.Errorf("recipient bech32: %w", err)
	}
	if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, rcpt, coins); err != nil {
		return fmt.Errorf("send to recipient: %w", err)
	}

	// Mark claimed
	k.setClaimed(ctx, intent.ProviderChainId, intent.EscrowId)

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