package mintburn

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/maany-xyz/maany-dex/v5/x/mintburn/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if gs == nil {
		gs = types.DefaultGenesis()
	}
	if err := gs.Validate(); err != nil {
		panic(err)
	}
	k.SetParams(ctx, gs.Params)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}
