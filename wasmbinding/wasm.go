package wasmbinding

import (
    wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
    bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	contractmanagerkeeper "github.com/maany-xyz/maany-dex/v5/x/contractmanager/keeper"
	cronkeeper "github.com/maany-xyz/maany-dex/v5/x/cron/keeper"
	feeburnerkeeper "github.com/maany-xyz/maany-dex/v5/x/feeburner/keeper"
	feerefunderkeeper "github.com/maany-xyz/maany-dex/v5/x/feerefunder/keeper"

	adminmodulekeeper "github.com/cosmos/admin-module/v2/x/adminmodule/keeper"

	marketmapkeeper "github.com/skip-mev/slinky/x/marketmap/keeper"
	oraclekeeper "github.com/skip-mev/slinky/x/oracle/keeper"

    interchainqueriesmodulekeeper "github.com/maany-xyz/maany-dex/v5/x/interchainqueries/keeper"
    interchaintransactionsmodulekeeper "github.com/maany-xyz/maany-dex/v5/x/interchaintxs/keeper"
)

// RegisterCustomPlugins returns wasmkeeper.Option that we can use to connect handlers for implemented custom queries and messages to the App
func RegisterCustomPlugins(
    ictxKeeper *interchaintransactionsmodulekeeper.Keeper,
    icqKeeper *interchainqueriesmodulekeeper.Keeper,
    // transfer transfer.KeeperTransferWrapper,
    adminKeeper *adminmodulekeeper.Keeper,
    feeBurnerKeeper *feeburnerkeeper.Keeper,
    feeRefunderKeeper *feerefunderkeeper.Keeper,
    bank *bankkeeper.BaseKeeper,
    cronKeeper *cronkeeper.Keeper,
    contractmanagerKeeper *contractmanagerkeeper.Keeper,
    oracleKeeper *oraclekeeper.Keeper,
    markemapKeeper *marketmapkeeper.Keeper,
) []wasmkeeper.Option {
    wasmQueryPlugin := NewQueryPlugin(ictxKeeper, icqKeeper, feeBurnerKeeper, feeRefunderKeeper, contractmanagerKeeper, oracleKeeper, markemapKeeper)

	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})
    messagePluginOpt := wasmkeeper.WithMessageHandlerDecorator(
        CustomMessageDecorator(ictxKeeper, icqKeeper, adminKeeper, bank, cronKeeper, contractmanagerKeeper),
    )

	return []wasmkeeper.Option{
		queryPluginOpt,
		messagePluginOpt,
	}
}
