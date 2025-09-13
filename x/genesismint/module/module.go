package genesismint

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/maany-xyz/maany-dex/v5/x/genesismint/keeper"
	"github.com/maany-xyz/maany-dex/v5/x/genesismint/types"
)

// Interface assertions for SDK v0.50+
var (
	_ module.HasServices     = (*AppModule)(nil)       // even if no services, ok to satisfy with no-op
	_ module.HasInvariants   = (*AppModule)(nil)       // optional; no-op ok
	_ module.HasConsensusVersion = (*AppModule)(nil)   // nice to have
)

/* -----------------------------
   AppModuleBasic
------------------------------*/

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return types.ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}
func (AppModuleBasic) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}

/* -----------------------------
   AppModule (marker + impls)
------------------------------*/

type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

// v0.50 marker methods:
func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}

// Optional but recommended for upgrades
func (am AppModule) ConsensusVersion() uint64 { return 1 }

// Services: no Msg/Query to register yet
func (am AppModule) RegisterServices(_ module.Configurator) {}

// Invariants: none
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

/* -----------------------------
   Genesis (ABCI form)
------------------------------*/

func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{cdc: cdc, keeper: k}
}

func (am AppModule) InitGenesis(
	ctx sdk.Context,
	cdc codec.JSONCodec,
	data json.RawMessage,
) []abci.ValidatorUpdate {
	var gen types.GenesisState
	cdc.MustUnmarshalJSON(data, &gen)

	if err := gen.Validate(); err != nil {
		panic(fmt.Errorf("genesismint genesis validate: %w", err))
	}

	// If you want to pre-mark claimed IDs, export a SetClaimed() on keeper and do it here.

	// IMPORTANT: strict verification at genesis requires the provider IBC client + consensus
	// state at >= proof height included in this genesis. Toggle strict=false if not available yet.
	strict := true
	for _, m := range gen.Mints {
		if err := am.keeper.ProcessGenesisMint(ctx, gen.Params, m, strict); err != nil {
			panic(fmt.Errorf("genesismint: escrow_id=%s: %w", m.EscrowId, err))
		}
	}
	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := types.GenesisState{
		Params:           &types.Params{},  // or load from KV if you persist params later
		Mints:            []*types.MintIntent{},
		ClaimedEscrowIds: []string{},
	}
	return cdc.MustMarshalJSON(&state)
}
