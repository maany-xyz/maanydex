package mintburn

import (
	"encoding/json"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	mintburn "github.com/maany-xyz/maany-dex/v5/x/mintburn/keeper"
	"github.com/maany-xyz/maany-dex/v5/x/mintburn/types"
)

var (
	_ module.AppModuleBasic = (*AppModuleBasic)(nil)
	_ module.AppModule      = (*AppModule)(nil)
	_ module.HasABCIGenesis = (*AppModule)(nil)
)

// -----------------------------
// AppModuleBasic
// -----------------------------

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (AppModuleBasic) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}

func (am AppModule) IsOnePerModuleType() {}

func (am AppModule) IsAppModule() {}

// DefaultGenesis returns the module's default genesis as raw JSON bytes (no proto needed).
func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesis())
	if err != nil {
		panic(err)
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return err
	}
	return gs.Validate()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// (Optional for CLI)
// func (AppModuleBasic) GetTxCmd() *cobra.Command    { return nil }
// func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

// -----------------------------
// AppModule
// -----------------------------

type AppModule struct {
	cdc    codec.Codec
	keeper mintburn.Keeper

	AppModuleBasic
}

func NewAppModule(cdc codec.Codec, k mintburn.Keeper) AppModule {
	return AppModule{
		cdc:            cdc,
		keeper:         k,
		AppModuleBasic: AppModuleBasic{},
	}
}

func (am AppModule) RegisterServices(_ module.Configurator) {}

// HasABCIGenesis: InitGenesis/ExportGenesis using JSON (no proto)
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(err)
	}
	am.keeper.InitGenesis(ctx, &gs)
	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	return bz
}

// Optional but recommended: advertise a consensus version for migrations.
func (am AppModule) ConsensusVersion() uint64 { return 1 }
