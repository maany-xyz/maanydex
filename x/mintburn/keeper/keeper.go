package mintburn

import (
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankKeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ---- Keep your IBC keepers the same ----

type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChannel string) (channeltypes.Channel, bool)
}

type ConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (connectiontypes.ConnectionEnd, bool)
}

type ClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (ibcexported.ClientState, bool)
}

// ---- Keeper ----

type Keeper struct {
	ModuleName     string
	StoreKey       storetypes.StoreKey
	BankKeeper     bankKeeper.Keeper
	ChannelKeeper  ChannelKeeper
	ConnectionKeeper ConnectionKeeper
	ClientKeeper     ClientKeeper
}

// Constructor
func NewKeeper(
	moduleName string,
	storeKey storetypes.StoreKey,
	bankKeeper bankKeeper.Keeper,
	channelKeeper ChannelKeeper,
	connectionKeeper ConnectionKeeper,
	clientKeeper ClientKeeper,
) Keeper {
	return Keeper{
		ModuleName:       moduleName,
		StoreKey:         storeKey,
		BankKeeper:       bankKeeper,
		ChannelKeeper:    channelKeeper,
		ConnectionKeeper: connectionKeeper,
		ClientKeeper:     clientKeeper,
	}
}

// Logger
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+k.ModuleName)
}

// ---------------------------------------------
// MintBurn v2 additions: proofs + mirror supply
// ---------------------------------------------

var (
	// proofID -> 1 (consumed)
	KeyProofPrefix = []byte("proof/")

	// string-encoded sdk.Int (total mirrored supply minted - burned)
	KeyMirrorSupply = []byte("mirror_supply")

	// optional params key (pause/admin/allowed denoms) if you add params later
	KeyParams = []byte("params")
)

// internal helpers
func (k Keeper) proofStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.StoreKey), KeyProofPrefix)
}

// HasProof returns true if the unique proof (packet identity) is already consumed.
func (k Keeper) HasProof(ctx sdk.Context, id []byte) bool {
	return k.proofStore(ctx).Has(id)
}

// ConsumeProof marks the proof as spent (prevents replay).
func (k Keeper) ConsumeProof(ctx sdk.Context, id []byte) {
	k.proofStore(ctx).Set(id, []byte{1})
}

// GetMirrorSupply returns the current mirrored supply (sdk.Int).
func (k Keeper) GetMirrorSupply(ctx sdk.Context) sdkmath.Int {
	bz := ctx.KVStore(k.StoreKey).Get(KeyMirrorSupply)
	if len(bz) == 0 {
		return sdkmath.ZeroInt()
	}
	intVal, ok := sdkmath.NewIntFromString(string(bz))
	if !ok {
		// defensive: if corrupted, treat as zero to avoid panics
		return sdkmath.ZeroInt()
	}
	return intVal
}

// AddMirrorSupply increments mirrored supply by amt (positive Int).
func (k Keeper) AddMirrorSupply(ctx sdk.Context, amt sdkmath.Int) {
	cur := k.GetMirrorSupply(ctx)
	next := cur.Add(amt)
	ctx.KVStore(k.StoreKey).Set(KeyMirrorSupply, []byte(next.String()))
}

// (optional) SubMirrorSupply if you need it for the return path later
func (k Keeper) SubMirrorSupply(ctx sdk.Context, amt sdkmath.Int) {
	cur := k.GetMirrorSupply(ctx)
	next := cur.Sub(amt)
	if next.IsNegative() {
		next = sdkmath.ZeroInt() // defensive clamp
	}
	ctx.KVStore(k.StoreKey).Set(KeyMirrorSupply, []byte(next.String()))
}

// ---------------------------------------------
// Existing functionality you still need on DEX
// ---------------------------------------------

// MintTokens mints native (mirrored) coins to the recipient on the DEX.
func (k Keeper) MintTokens(ctx sdk.Context, recipient sdk.AccAddress, amount sdk.Coin) error {
	ctx.Logger().Info("mintburn: MintTokens")
	if err := k.BankKeeper.MintCoins(ctx, k.ModuleName, sdk.NewCoins(amount)); err != nil {
		return err
	}
	return k.BankKeeper.SendCoinsFromModuleToAccount(ctx, k.ModuleName, recipient, sdk.NewCoins(amount))
}

// IsAllowedChannel checks the allowlist (populated by middleware on channel open).
func (k Keeper) IsAllowedChannel(ctx sdk.Context, channelID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.StoreKey), []byte("allowed-channel/"))
	return store.Has([]byte(channelID))
}

func (k Keeper) BurnEscrowedTokens(ctx sdk.Context, escrowAddr sdk.AccAddress, coin sdk.Coin) error {
    	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, escrowAddr, k.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}

	return k.BankKeeper.BurnCoins(ctx, k.ModuleName, sdk.NewCoins(coin))
}

// ---------------------------------------------
// Removed / not needed on DEX in v2
// ---------------------------------------------
//
// - BurnTokens(...)                // not used on the DEX in the provider→DEX direction
// - BurnEscrowedTokens(...)        // escrow belongs to ICS-20 on the provider side
// - GetBalances(...)               // was only a debug helper; keep it locally if you like
//
// If you still want GetBalances for debugging, keep it; otherwise it’s safe to remove.
// For clarity, I’ve rem
