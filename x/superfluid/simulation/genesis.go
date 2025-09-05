package simulation

import (
	"encoding/json"
	"fmt"

	"github.com/maany-xyz/maany-dex/v5/osmomath"
	"github.com/maany-xyz/maany-dex/v5/x/superfluid/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// RandomizedGenState generates a random GenesisState for staking.
func RandomizedGenState(simState *module.SimulationState) {
	superfluidGenesis := &types.GenesisState{
		Params: types.Params{
			MinimumRiskFactor: osmomath.NewDecWithPrec(5, 2), // 5%
		},
		SuperfluidAssets:          []types.SuperfluidAsset{},
		OsmoEquivalentMultipliers: []types.OsmoEquivalentMultiplierRecord{},
	}

	bz, err := json.MarshalIndent(&superfluidGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated superfluid parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(superfluidGenesis)
}
