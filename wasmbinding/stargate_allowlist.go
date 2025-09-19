package wasmbinding

import (
    wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
    banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
    icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
    ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
    ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types" //nolint:staticcheck
    ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
    ibcchanneltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
    consumertypes "github.com/cosmos/interchain-security/v5/x/ccv/consumer/types"
    globalfeetypes "github.com/maany-xyz/maany-dex/v5/x/globalfee/types"
    feemarkettypes "github.com/skip-mev/feemarket/x/feemarket/types"
    marketmaptypes "github.com/skip-mev/slinky/x/marketmap/types"
    oracletypes "github.com/skip-mev/slinky/x/oracle/types"
    crontypes "github.com/maany-xyz/maany-dex/v5/x/cron/types"
    feeburnertypes "github.com/maany-xyz/maany-dex/v5/x/feeburner/types"
    interchainqueriestypes "github.com/maany-xyz/maany-dex/v5/x/interchainqueries/types"
    interchaintxstypes "github.com/maany-xyz/maany-dex/v5/x/interchaintxs/types"
)

func AcceptedStargateQueries() wasmkeeper.AcceptedQueries {
	return wasmkeeper.AcceptedQueries{
		// ibc
		"/ibc.core.client.v1.Query/ClientState":         &ibcclienttypes.QueryClientStateResponse{},
		"/ibc.core.client.v1.Query/ConsensusState":      &ibcclienttypes.QueryConsensusStateResponse{},
		"/ibc.core.connection.v1.Query/Connection":      &ibcconnectiontypes.QueryConnectionResponse{},
		"/ibc.core.channel.v1.Query/ChannelClientState": &ibcchanneltypes.QueryChannelClientStateResponse{},

        // interchain accounts
        "/ibc.applications.interchain_accounts.controller.v1.Query/InterchainAccount": &icacontrollertypes.QueryInterchainAccountResponse{},

		// transfer
		"/ibc.applications.transfer.v1.Query/DenomTrace":    &ibctransfertypes.QueryDenomTraceResponse{},
		"/ibc.applications.transfer.v1.Query/EscrowAddress": &ibctransfertypes.QueryEscrowAddressResponse{},

		// auth
		"/cosmos.auth.v1beta1.Query/Account": &authtypes.QueryAccountResponse{},
		"/cosmos.auth.v1beta1.Query/Params":  &authtypes.QueryParamsResponse{},

		// bank
		"/cosmos.bank.v1beta1.Query/Balance":       &banktypes.QueryBalanceResponse{},
		"/cosmos.bank.v1beta1.Query/DenomMetadata": &banktypes.QueryDenomsMetadataResponse{},
		"/cosmos.bank.v1beta1.Query/Params":        &banktypes.QueryParamsResponse{},
		"/cosmos.bank.v1beta1.Query/SupplyOf":      &banktypes.QuerySupplyOfResponse{},

		// interchaintxs
		"/neutron.interchaintxs.v1.Query/Params":                   &interchaintxstypes.QueryParamsResponse{},
		"/neutron.interchaintxs.v1.Query/InterchainAccountAddress": &interchaintxstypes.QueryInterchainAccountAddressResponse{},

		// cron
		"/neutron.cron.Query/Params": &crontypes.QueryParamsResponse{},

		// interchainqueries
		"/neutron.interchainqueries.Query/Params":            &interchainqueriestypes.QueryParamsResponse{},
		"/neutron.interchainqueries.Query/RegisteredQueries": &interchainqueriestypes.QueryRegisteredQueriesResponse{},
		"/neutron.interchainqueries.Query/RegisteredQuery":   &interchainqueriestypes.QueryRegisteredQueryResponse{},
		"/neutron.interchainqueries.Query/QueryResult":       &interchainqueriestypes.QueryRegisteredQueryResultResponse{},
		"/neutron.interchainqueries.Query/LastRemoteHeight":  &interchainqueriestypes.QueryLastRemoteHeightResponse{},

		// feeburner
		"/neutron.feeburner.Query/Params":                    &feeburnertypes.QueryParamsResponse{},
		"/neutron.feeburner.Query/TotalBurnedNeutronsAmount": &feeburnertypes.QueryTotalBurnedNeutronsAmountResponse{},

		// dex

		// oracle
		"/slinky.oracle.v1.Query/GetAllCurrencyPairs": &oracletypes.GetAllCurrencyPairsResponse{},
		"/slinky.oracle.v1.Query/GetPrice":            &oracletypes.GetPriceResponse{},
		"/slinky.oracle.v1.Query/GetPrices":           &oracletypes.GetPricesResponse{},

		// marketmap
		"/slinky.marketmap.v1.Query/MarketMap":   &marketmaptypes.MarketMapResponse{},
		"/slinky.marketmap.v1.Query/LastUpdated": &marketmaptypes.LastUpdatedResponse{},
		"/slinky.marketmap.v1.Query/Params":      &marketmaptypes.ParamsResponse{},
		"/slinky.marketmap.v1.Query/Market":      &marketmaptypes.MarketResponse{},

		// feemarket
		"/feemarket.feemarket.v1.Query/Params":    &feemarkettypes.ParamsResponse{},
		"/feemarket.feemarket.v1.Query/State":     &feemarkettypes.StateResponse{},
		"/feemarket.feemarket.v1.Query/GasPrice":  &feemarkettypes.GasPriceResponse{},
		"/feemarket.feemarket.v1.Query/GasPrices": &feemarkettypes.GasPricesResponse{},

        // globalfee
        "/gaia.globalfee.v1beta1.Query/Params": &globalfeetypes.QueryParamsResponse{},

		// consumer
		"/interchain_security.ccv.consumer.v1.Query/QueryParams": &consumertypes.QueryParamsResponse{},
	}
}
