package wasmbinding

import (
    contractmanagerkeeper "github.com/maany-xyz/maany-dex/v5/x/contractmanager/keeper"
    contractmanagertypes "github.com/maany-xyz/maany-dex/v5/x/contractmanager/types"
    feeburnerkeeper "github.com/maany-xyz/maany-dex/v5/x/feeburner/keeper"
    feerefunderkeeper "github.com/maany-xyz/maany-dex/v5/x/feerefunder/keeper"
    icqkeeper "github.com/maany-xyz/maany-dex/v5/x/interchainqueries/keeper"
    icacontrollerkeeper "github.com/maany-xyz/maany-dex/v5/x/interchaintxs/keeper"
    marketmapkeeper "github.com/skip-mev/slinky/x/marketmap/keeper"
    oraclekeeper "github.com/skip-mev/slinky/x/oracle/keeper"
)

type QueryPlugin struct {
    icaControllerKeeper        *icacontrollerkeeper.Keeper
    icqKeeper                  *icqkeeper.Keeper
    feeBurnerKeeper            *feeburnerkeeper.Keeper
    feeRefunderKeeper          *feerefunderkeeper.Keeper
    contractmanagerQueryServer contractmanagertypes.QueryServer
    oracleKeeper               *oraclekeeper.Keeper
    marketmapKeeper            *marketmapkeeper.Keeper
}

// NewQueryPlugin returns a reference to a new QueryPlugin.
func NewQueryPlugin(icaControllerKeeper *icacontrollerkeeper.Keeper, icqKeeper *icqkeeper.Keeper, feeBurnerKeeper *feeburnerkeeper.Keeper, feeRefunderKeeper *feerefunderkeeper.Keeper, contractmanagerKeeper *contractmanagerkeeper.Keeper, oracleKeeper *oraclekeeper.Keeper, marketmapKeeper *marketmapkeeper.Keeper) *QueryPlugin {
    return &QueryPlugin{
        icaControllerKeeper:        icaControllerKeeper,
        icqKeeper:                  icqKeeper,
        feeBurnerKeeper:            feeBurnerKeeper,
        feeRefunderKeeper:          feeRefunderKeeper,
        contractmanagerQueryServer: contractmanagerkeeper.NewQueryServerImpl(*contractmanagerKeeper),
        oracleKeeper:               oracleKeeper,
        marketmapKeeper:            marketmapKeeper,
    }
}
