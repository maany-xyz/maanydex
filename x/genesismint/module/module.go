package genesismint

import (
    "context"
    "encoding/json"
    "fmt"
    stdjson "encoding/json"

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
    _ appmodule.HasBeginBlocker  = AppModule{}
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
    // 1) First, strip unsupported (non-proto) ICA config keys from params so proto unmarshal won't error.
    sanitized := data
    var extractedConn, extractedOwner string
    var extractedTO uint64
    var extractedMC int
    var haveConn, haveOwner, haveTO, haveMC bool

    var raw map[string]stdjson.RawMessage
    if err := stdjson.Unmarshal(data, &raw); err == nil {
        if pRaw, ok := raw["params"]; ok {
            var pMap map[string]any
            if err := stdjson.Unmarshal(pRaw, &pMap); err == nil {
                // ica_controller_connection_id / icaControllerConnectionId
                if s, ok := pMap["ica_controller_connection_id"].(string); ok && s != "" {
                    extractedConn, haveConn = s, true
                    delete(pMap, "ica_controller_connection_id")
                } else if s, ok := pMap["icaControllerConnectionId"].(string); ok && s != "" {
                    extractedConn, haveConn = s, true
                    delete(pMap, "icaControllerConnectionId")
                }
                // ica_owner / icaOwner
                if s, ok := pMap["ica_owner"].(string); ok && s != "" {
                    extractedOwner, haveOwner = s, true
                    delete(pMap, "ica_owner")
                } else if s, ok := pMap["icaOwner"].(string); ok && s != "" {
                    extractedOwner, haveOwner = s, true
                    delete(pMap, "icaOwner")
                }
                // ica_tx_timeout_seconds / icaTxTimeoutSeconds
                if v, ok := pMap["ica_tx_timeout_seconds"]; ok {
                    switch t := v.(type) {
                    case float64:
                        extractedTO, haveTO = uint64(t), true
                    case string:
                        var u uint64; if _, err := fmt.Sscan(t, &u); err == nil { extractedTO, haveTO = u, true }
                    }
                    delete(pMap, "ica_tx_timeout_seconds")
                } else if v, ok := pMap["icaTxTimeoutSeconds"]; ok {
                    switch t := v.(type) {
                    case float64:
                        extractedTO, haveTO = uint64(t), true
                    case string:
                        var u uint64; if _, err := fmt.Sscan(t, &u); err == nil { extractedTO, haveTO = u, true }
                    }
                    delete(pMap, "icaTxTimeoutSeconds")
                }
                // ica_max_claims_per_block / icaMaxClaimsPerBlock / icaMaxClaimPerBlock
                if v, ok := pMap["ica_max_claims_per_block"]; ok {
                    switch t := v.(type) {
                    case float64:
                        extractedMC, haveMC = int(t), true
                    case string:
                        var u int; if _, err := fmt.Sscan(t, &u); err == nil { extractedMC, haveMC = u, true }
                    }
                    delete(pMap, "ica_max_claims_per_block")
                } else if v, ok := pMap["icaMaxClaimsPerBlock"]; ok {
                    switch t := v.(type) {
                    case float64:
                        extractedMC, haveMC = int(t), true
                    case string:
                        var u int; if _, err := fmt.Sscan(t, &u); err == nil { extractedMC, haveMC = u, true }
                    }
                    delete(pMap, "icaMaxClaimsPerBlock")
                } else if v, ok := pMap["icaMaxClaimPerBlock"]; ok {
                    switch t := v.(type) {
                    case float64:
                        extractedMC, haveMC = int(t), true
                    case string:
                        var u int; if _, err := fmt.Sscan(t, &u); err == nil { extractedMC, haveMC = u, true }
                    }
                    delete(pMap, "icaMaxClaimPerBlock")
                }

                // re-pack sanitized params back into raw and overall genesis
                if sanitizedParams, err := stdjson.Marshal(pMap); err == nil {
                    raw["params"] = sanitizedParams
                    if sanitizedAll, err := stdjson.Marshal(raw); err == nil {
                        sanitized = sanitizedAll
                    }
                }
            }
        }
    }

    // 2) Now proto-unmarshal the sanitized JSON into our typed genesis state.
    var gs types.GenesisState
    cdc.MustUnmarshalJSON(sanitized, &gs)

    // ✅ validate here since HasGenesis isn't asserted
    if err := gs.Validate(); err != nil {
        panic(fmt.Errorf("genesismint genesis validate: %w", err))
    }

    // 3) Apply any extracted ICA config values.
    if haveConn { am.keeper.SetICAConnectionID(ctx, extractedConn) }
    if haveOwner { am.keeper.SetICAOwner(ctx, extractedOwner) }
    if haveTO { am.keeper.SetICATimeoutSeconds(ctx, extractedTO) }
    if haveMC { am.keeper.SetMaxClaimsPerBlock(ctx, extractedMC) }

    strict := true // set false if the provider client+consensus@height isn't in genesis yet
    ctx.Logger().Info("genesismint: InitGenesis start",
        "mint_count", len(gs.Mints),
        "strict_proof", strict,
    )
    for _, m := range gs.Mints {
        if err := am.keeper.ProcessGenesisMint(ctx, gs.Params, m, strict); err != nil {
            panic(fmt.Errorf("genesismint: escrow_id=%s: %w", m.EscrowId, err))
        }
        ctx.Logger().Info("genesismint: processed mint intent", "escrow_id", m.EscrowId)
    }
    ctx.Logger().Info("genesismint: InitGenesis done")
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

// ABCI hooks (v0.50 style)
func (am AppModule) BeginBlock(ctx context.Context) error {
    am.keeper.BeginBlocker(sdk.UnwrapSDKContext(ctx))
    return nil
}
