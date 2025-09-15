package genesismint

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// If your project uses gateway v1, replace the next line with:
	//   runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/maany-xyz/maany-dex/v5/x/genesismint/keeper"
	"github.com/maany-xyz/maany-dex/v5/x/genesismint/types"
)

/* ===== Interface assertions (SDK v0.50) ===== */

var (
	_ appmodule.AppModule         = AppModule{} // marker + Name()
	_ module.HasABCIGenesis      = (*AppModule)(nil) // InitGenesis + ExportGenesis
	_ module.HasServices         = (*AppModule)(nil)
	_ module.HasInvariants       = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)
)

/* ===== AppModuleBasic ===== */

type AppModuleBasic struct {}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (am AppModule) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (am AppModule) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (am AppModule) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}
/* ===== AppModule ===== */

type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{cdc: cdc, keeper: k}
}

// Some SDKs still require Name() on AppModule too.
func (am AppModule) Name() string { return types.ModuleName }

// v0.50 marker methods
func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}

func (am AppModule) ConsensusVersion() uint64 { return 1 }

// No Msg/Query services yet; keep as no-op
func (am AppModule) RegisterServices(_ module.Configurator) {}

// No invariants
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

/* ===== Genesis (split across HasGenesis + HasABCIGenesis) ===== */

// HasGenesis: Default + Validate (NOTE: new Validate signature includes txCfg)
func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// SDK v0.50 expects (cdc, txCfg, data) — no ctx here.
func (am AppModule) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, data json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		return fmt.Errorf("unmarshal genesismint genesis: %w", err)
	}
	return gs.Validate()
}

func (am AppModule) InitGenesis(
	ctx sdk.Context,
	cdc codec.JSONCodec,
	data json.RawMessage,
) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)

	// ✅ validate here since HasGenesis isn't asserted
	if err := gs.Validate(); err != nil {
		panic(fmt.Errorf("genesismint genesis validate: %w", err))
	}

	strict := true // set false if the provider client+consensus@height isn't in genesis yet
	for _, m := range gs.Mints {
		if err := am.keeper.ProcessGenesisMint(ctx, gs.Params, m, strict); err != nil {
			panic(fmt.Errorf("genesismint: escrow_id=%s: %w", m.EscrowId, err))
		}
	}
	return nil
}
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := types.GenesisState{
		Params:           &types.Params{},
		Mints:            []*types.MintIntent{},
		ClaimedEscrowIds: []string{},
	}
	return cdc.MustMarshalJSON(&state)
}
