package wasmbinding

import (
	"encoding/json"
	"fmt"

	contractmanagerkeeper "github.com/maany-xyz/maany-dex/v5/x/contractmanager/keeper"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"cosmossdk.io/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	crontypes "github.com/maany-xyz/maany-dex/v5/x/cron/types"

	cronkeeper "github.com/maany-xyz/maany-dex/v5/x/cron/keeper"

	paramChange "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	adminmodulekeeper "github.com/cosmos/admin-module/v2/x/adminmodule/keeper"
	admintypes "github.com/cosmos/admin-module/v2/x/adminmodule/types"

	contractmanagertypes "github.com/maany-xyz/maany-dex/v5/x/contractmanager/types"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	//nolint:staticcheck

	"github.com/maany-xyz/maany-dex/v5/wasmbinding/bindings"
	icqkeeper "github.com/maany-xyz/maany-dex/v5/x/interchainqueries/keeper"
	icqtypes "github.com/maany-xyz/maany-dex/v5/x/interchainqueries/types"
	ictxkeeper "github.com/maany-xyz/maany-dex/v5/x/interchaintxs/keeper"
	ictxtypes "github.com/maany-xyz/maany-dex/v5/x/interchaintxs/types"

    
)

func CustomMessageDecorator(
    ictx *ictxkeeper.Keeper,
    icq *icqkeeper.Keeper,
    adminKeeper *adminmodulekeeper.Keeper,
    bankKeeper *bankkeeper.BaseKeeper,
    cronKeeper *cronkeeper.Keeper,
    contractmanagerKeeper *contractmanagerkeeper.Keeper,
) func(messenger wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(old wasmkeeper.Messenger) wasmkeeper.Messenger {
		return &CustomMessenger{
			Keeper:                     *ictx,
			Wrapped:                    old,
			Ictxmsgserver:              ictxkeeper.NewMsgServerImpl(*ictx),
			Icqmsgserver:               icqkeeper.NewMsgServerImpl(*icq),
            Adminserver:                adminmodulekeeper.NewMsgServerImpl(*adminKeeper),
            Bank:                       bankKeeper,
            CronMsgServer:              cronkeeper.NewMsgServerImpl(*cronKeeper),
            CronQueryServer:            cronKeeper,
            AdminKeeper:                adminKeeper,
            ContractmanagerMsgServer:   contractmanagerkeeper.NewMsgServerImpl(*contractmanagerKeeper),
            ContractmanagerQueryServer: contractmanagerkeeper.NewQueryServerImpl(*contractmanagerKeeper),
        }
    }
}

type CustomMessenger struct {
	Keeper                     ictxkeeper.Keeper
	Wrapped                    wasmkeeper.Messenger
	Ictxmsgserver              ictxtypes.MsgServer
	Icqmsgserver               icqtypes.MsgServer
	Adminserver                admintypes.MsgServer
	Bank                       *bankkeeper.BaseKeeper
    CronMsgServer              crontypes.MsgServer
    CronQueryServer            crontypes.QueryServer
	AdminKeeper                *adminmodulekeeper.Keeper
	ContractmanagerMsgServer   contractmanagertypes.MsgServer
	ContractmanagerQueryServer contractmanagertypes.QueryServer
}

var _ wasmkeeper.Messenger = (*CustomMessenger)(nil)

func (m *CustomMessenger) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	// Return early if msg.Custom is nil
	if msg.Custom == nil {
		return m.Wrapped.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
	}

	var contractMsg bindings.NeutronMsg
	if err := json.Unmarshal(msg.Custom, &contractMsg); err != nil {
		ctx.Logger().Debug("json.Unmarshal: failed to decode incoming custom cosmos message",
			"from_address", contractAddr.String(),
			"message", string(msg.Custom),
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to decode incoming custom cosmos message")
	}

	// Dispatch the message based on its type by checking each possible field
	if contractMsg.SubmitTx != nil {
		return m.submitTx(ctx, contractAddr, contractMsg.SubmitTx)
	}
	if contractMsg.RegisterInterchainAccount != nil {
		return m.registerInterchainAccount(ctx, contractAddr, contractMsg.RegisterInterchainAccount)
	}
	if contractMsg.RegisterInterchainQuery != nil {
		return m.registerInterchainQuery(ctx, contractAddr, contractMsg.RegisterInterchainQuery)
	}
	if contractMsg.UpdateInterchainQuery != nil {
		return m.updateInterchainQuery(ctx, contractAddr, contractMsg.UpdateInterchainQuery)
	}
	if contractMsg.RemoveInterchainQuery != nil {
		return m.removeInterchainQuery(ctx, contractAddr, contractMsg.RemoveInterchainQuery)
	}
	// if contractMsg.IBCTransfer != nil {
	// 	return m.ibcTransfer(ctx, contractAddr, *contractMsg.IBCTransfer)
	// }
	if contractMsg.SubmitAdminProposal != nil {
		return m.submitAdminProposal(ctx, contractAddr, &contractMsg.SubmitAdminProposal.AdminProposal)
	}

    // message handlers for removed modules omitted

	if contractMsg.AddSchedule != nil {
		return m.addSchedule(ctx, contractAddr, contractMsg.AddSchedule)
	}
	if contractMsg.RemoveSchedule != nil {
		return m.removeSchedule(ctx, contractAddr, contractMsg.RemoveSchedule)
	}
	if contractMsg.ResubmitFailure != nil {
		return m.resubmitFailure(ctx, contractAddr, contractMsg.ResubmitFailure)
	}

	// If none of the conditions are met, forward the message to the wrapped handler
	return m.Wrapped.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
}

// func handleDexMsg[T sdk.LegacyMsg, R proto.Message](ctx sdk.Context, msg T, handler func(ctx context.Context, msg T) (R, error)) ([][]byte, [][]*types.Any, error) {
// 	if len(msg.GetSigners()) != 1 {
// 		// should never happen
// 		panic("should be 1 signer")
// 	}
// 	signer := msg.GetSigners()[0].String()

// 	resp, err := handler(ctx, msg)
// 	if err != nil {
// 		ctx.Logger().Debug(fmt.Sprintf("%T: failed to execute", msg),
// 			"from_address", signer,
// 			"msg", msg,
// 			"error", err,
// 		)
// 		return nil, nil, errors.Wrapf(err, "failed to execute %T", msg)
// 	}

// 	data, err := json.Marshal(resp)
// 	if err != nil {
// 		ctx.Logger().Error(fmt.Sprintf("json.Marshal: failed to marshal %T response to JSON", resp),
// 			"from_address", signer,
// 			"error", err,
// 		)
// 		return nil, nil, errors.Wrap(err, fmt.Sprintf("marshal %T failed", resp))
// 	}

// 	ctx.Logger().Debug(fmt.Sprintf("%T execution completed", msg),
// 		"from_address", signer,
// 		"msg", msg,
// 	)

// 	anyResp, err := types.NewAnyWithValue(resp)
// 	if err != nil {
// 		return nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", resp)
// 	}
// 	msgResponses := [][]*types.Any{{anyResp}}
// 	return [][]byte{data}, msgResponses, nil
// }

// func (m *CustomMessenger) dispatchDexMsg(ctx sdk.Context, contractAddr sdk.AccAddress, dex bindings.Dex) ([][]byte, [][]*types.Any, error) {
// 	switch {
// 	case dex.Deposit != nil:
// 		dex.Deposit.Creator = contractAddr.String()
// 		return handleDexMsg(ctx, dex.Deposit, m.DexMsgServer.Deposit)
// 	case dex.Withdrawal != nil:
// 		dex.Withdrawal.Creator = contractAddr.String()
// 		return handleDexMsg(ctx, dex.Withdrawal, m.DexMsgServer.Withdrawal)
// 	case dex.PlaceLimitOrder != nil:
// 		msg := dextypes.MsgPlaceLimitOrder{
// 			Creator:  contractAddr.String(),
// 			Receiver: dex.PlaceLimitOrder.Receiver,
// 			TokenIn:  dex.PlaceLimitOrder.TokenIn,
// 			TokenOut: dex.PlaceLimitOrder.TokenOut,
// 			//nolint: staticcheck // TODO: remove in next release
// 			TickIndexInToOut: dex.PlaceLimitOrder.TickIndexInToOut,
// 			AmountIn:         dex.PlaceLimitOrder.AmountIn,
// 			MaxAmountOut:     dex.PlaceLimitOrder.MaxAmountOut,
// 		}
// 		orderTypeInt, ok := dextypes.LimitOrderType_value[dex.PlaceLimitOrder.OrderType]
// 		if !ok {
// 			return nil, nil, errors.Wrap(dextypes.ErrInvalidOrderType,
// 				fmt.Sprintf(
// 					"got \"%s\", expected one of %s",
// 					dex.PlaceLimitOrder.OrderType,
// 					strings.Join(maps.Keys(dextypes.LimitOrderType_value), ", ")),
// 			)
// 		}
// 		msg.OrderType = dextypes.LimitOrderType(orderTypeInt)

// 		if dex.PlaceLimitOrder.ExpirationTime != nil {
// 			t := time.Unix(int64(*(dex.PlaceLimitOrder.ExpirationTime)), 0)
// 			msg.ExpirationTime = &t
// 		}

// 		if limitPriceStr := dex.PlaceLimitOrder.LimitSellPrice; limitPriceStr != "" {
// 			limitPriceDec, err := dexutils.ParsePrecDecScientificNotation(limitPriceStr)
// 			if err != nil {
// 				return nil, nil, errors.Wrapf(err, "cannot parse string %s for limit price", limitPriceStr)
// 			}
// 			msg.LimitSellPrice = &limitPriceDec
// 		}

// 		return handleDexMsg(ctx, &msg, m.DexMsgServer.PlaceLimitOrder)
// 	case dex.CancelLimitOrder != nil:
// 		dex.CancelLimitOrder.Creator = contractAddr.String()
// 		return handleDexMsg(ctx, dex.CancelLimitOrder, m.DexMsgServer.CancelLimitOrder)
// 	case dex.WithdrawFilledLimitOrder != nil:
// 		dex.WithdrawFilledLimitOrder.Creator = contractAddr.String()
// 		return handleDexMsg(ctx, dex.WithdrawFilledLimitOrder, m.DexMsgServer.WithdrawFilledLimitOrder)
// 	case dex.MultiHopSwap != nil:
// 		dex.MultiHopSwap.Creator = contractAddr.String()
// 		return handleDexMsg(ctx, dex.MultiHopSwap, m.DexMsgServer.MultiHopSwap)
// 	}

// 	return nil, nil, sdkerrors.ErrUnknownRequest
// }

// func (m *CustomMessenger) ibcTransfer(ctx sdk.Context, contractAddr sdk.AccAddress, ibcTransferMsg transferwrappertypes.MsgTransfer) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
// 	ibcTransferMsg.Sender = contractAddr.String()

// 	// response, err := m.transferKeeper.Transfer(ctx, &ibcTransferMsg)
// 	// if err != nil {
// 	// 	ctx.Logger().Debug("transferServer.Transfer: failed to transfer",
// 	// 		"from_address", contractAddr.String(),
// 	// 		"msg", ibcTransferMsg,
// 	// 		"error", err,
// 	// 	)
// 	// 	return nil, nil, nil, errors.Wrap(err, "failed to execute IBCTransfer")
// 	// }

// 	data, err := json.Marshal(response)
// 	if err != nil {
// 		ctx.Logger().Error("json.Marshal: failed to marshal MsgTransferResponse response to JSON",
// 			"from_address", contractAddr.String(),
// 			"msg", response,
// 			"error", err,
// 		)
// 		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
// 	}

// 	ctx.Logger().Debug("ibcTransferMsg completed",
// 		"from_address", contractAddr.String(),
// 		"msg", ibcTransferMsg,
// 	)

// 	anyResp, err := types.NewAnyWithValue(response)
// 	if err != nil {
// 		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", response)
// 	}
// 	msgResponses := [][]*types.Any{{anyResp}}
// 	return nil, [][]byte{data}, msgResponses, nil
// }

func (m *CustomMessenger) updateInterchainQuery(ctx sdk.Context, contractAddr sdk.AccAddress, updateQuery *bindings.UpdateInterchainQuery) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	response, err := m.performUpdateInterchainQuery(ctx, contractAddr, updateQuery)
	if err != nil {
		ctx.Logger().Debug("performUpdateInterchainQuery: failed to update interchain query",
			"from_address", contractAddr.String(),
			"msg", updateQuery,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to update interchain query")
	}

	data, err := json.Marshal(response)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal UpdateInterchainQueryResponse response to JSON",
			"from_address", contractAddr.String(),
			"msg", updateQuery,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	ctx.Logger().Debug("interchain query updated",
		"from_address", contractAddr.String(),
		"msg", updateQuery,
	)

	anyResp, err := types.NewAnyWithValue(response)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", response)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) performUpdateInterchainQuery(ctx sdk.Context, contractAddr sdk.AccAddress, updateQuery *bindings.UpdateInterchainQuery) (*icqtypes.MsgUpdateInterchainQueryResponse, error) {
	msg := icqtypes.MsgUpdateInterchainQueryRequest{
		QueryId:               updateQuery.QueryId,
		NewKeys:               updateQuery.NewKeys,
		NewUpdatePeriod:       updateQuery.NewUpdatePeriod,
		NewTransactionsFilter: updateQuery.NewTransactionsFilter,
		Sender:                contractAddr.String(),
	}

	response, err := m.Icqmsgserver.UpdateInterchainQuery(ctx, &msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update interchain query")
	}

	return response, nil
}

func (m *CustomMessenger) removeInterchainQuery(ctx sdk.Context, contractAddr sdk.AccAddress, removeQuery *bindings.RemoveInterchainQuery) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	response, err := m.performRemoveInterchainQuery(ctx, contractAddr, removeQuery)
	if err != nil {
		ctx.Logger().Debug("performRemoveInterchainQuery: failed to update interchain query",
			"from_address", contractAddr.String(),
			"msg", removeQuery,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to remove interchain query")
	}

	data, err := json.Marshal(response)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal RemoveInterchainQueryResponse response to JSON",
			"from_address", contractAddr.String(),
			"msg", removeQuery,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	ctx.Logger().Debug("interchain query removed",
		"from_address", contractAddr.String(),
		"msg", removeQuery,
	)

	anyResp, err := types.NewAnyWithValue(response)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", response)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) performRemoveInterchainQuery(ctx sdk.Context, contractAddr sdk.AccAddress, updateQuery *bindings.RemoveInterchainQuery) (*icqtypes.MsgRemoveInterchainQueryResponse, error) {
	msg := icqtypes.MsgRemoveInterchainQueryRequest{
		QueryId: updateQuery.QueryId,
		Sender:  contractAddr.String(),
	}

	response, err := m.Icqmsgserver.RemoveInterchainQuery(ctx, &msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to remove interchain query")
	}

	return response, nil
}

func (m *CustomMessenger) submitTx(ctx sdk.Context, contractAddr sdk.AccAddress, submitTx *bindings.SubmitTx) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	response, err := m.performSubmitTx(ctx, contractAddr, submitTx)
	if err != nil {
		ctx.Logger().Debug("performSubmitTx: failed to submit interchain transaction",
			"from_address", contractAddr.String(),
			"connection_id", submitTx.ConnectionId,
			"interchain_account_id", submitTx.InterchainAccountId,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to submit interchain transaction")
	}

	data, err := json.Marshal(response)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal submitTx response to JSON",
			"from_address", contractAddr.String(),
			"connection_id", submitTx.ConnectionId,
			"interchain_account_id", submitTx.InterchainAccountId,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	ctx.Logger().Debug("interchain transaction submitted",
		"from_address", contractAddr.String(),
		"connection_id", submitTx.ConnectionId,
		"interchain_account_id", submitTx.InterchainAccountId,
	)

	anyResp, err := types.NewAnyWithValue(response)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", response)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) submitAdminProposal(ctx sdk.Context, contractAddr sdk.AccAddress, adminProposal *bindings.AdminProposal) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	var data []byte
	err := m.validateProposalQty(adminProposal)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "invalid proposal quantity")
	}
	// here we handle pre-v2.0.0 style of proposals: param change, upgrade, client update
	if m.isLegacyProposal(adminProposal) {
		resp, err := m.performSubmitAdminProposalLegacy(ctx, contractAddr, adminProposal)
		if err != nil {
			ctx.Logger().Debug("performSubmitAdminProposalLegacy: failed to submitAdminProposal",
				"from_address", contractAddr.String(),
				"error", err,
			)
			return nil, nil, nil, errors.Wrap(err, "failed to submit admin proposal legacy")
		}
		data, err = json.Marshal(resp)
		if err != nil {
			ctx.Logger().Error("json.Marshal: failed to marshal submitAdminProposalLegacy response to JSON",
				"from_address", contractAddr.String(),
				"error", err,
			)
			return nil, nil, nil, errors.Wrap(err, "marshal json failed")
		}

		ctx.Logger().Debug("submit proposal legacy submitted",
			"from_address", contractAddr.String(),
		)

		anyResp, err := types.NewAnyWithValue(resp)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", resp)
		}
		msgResponses := [][]*types.Any{{anyResp}}
		return nil, [][]byte{data}, msgResponses, nil
	}

	resp, err := m.performSubmitAdminProposal(ctx, contractAddr, adminProposal)
	if err != nil {
		ctx.Logger().Debug("performSubmitAdminProposal: failed to submitAdminProposal",
			"from_address", contractAddr.String(),
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to submit admin proposal")
	}

	data, err = json.Marshal(resp)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal submitAdminProposal response to JSON",
			"from_address", contractAddr.String(),
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	ctx.Logger().Debug("submit proposal message submitted",
		"from_address", contractAddr.String(),
	)

	anyResp, err := types.NewAnyWithValue(resp)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", resp)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) performSubmitAdminProposalLegacy(ctx sdk.Context, contractAddr sdk.AccAddress, adminProposal *bindings.AdminProposal) (*admintypes.MsgSubmitProposalLegacyResponse, error) {
	proposal := adminProposal
	msg := admintypes.MsgSubmitProposalLegacy{Proposer: contractAddr.String()}

	switch {
	case proposal.ParamChangeProposal != nil:
		p := proposal.ParamChangeProposal
		err := msg.SetContent(&paramChange.ParameterChangeProposal{
			Title:       p.Title,
			Description: p.Description,
			Changes:     p.ParamChanges,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to set content on ParameterChangeProposal")
		}
	default:
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "unexpected legacy admin proposal structure: %+v", proposal)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, errors.Wrap(err, "failed to validate incoming SubmitAdminProposal message")
	}

	response, err := m.Adminserver.SubmitProposalLegacy(ctx, &msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to submit proposal")
	}

	ctx.Logger().Debug("submit proposal legacy processed in msg server",
		"from_address", contractAddr.String(),
	)

	return response, nil
}

func (m *CustomMessenger) performSubmitAdminProposal(ctx sdk.Context, contractAddr sdk.AccAddress, adminProposal *bindings.AdminProposal) (*admintypes.MsgSubmitProposalResponse, error) {
	proposal := adminProposal
	authority := authtypes.NewModuleAddress(admintypes.ModuleName)
	var (
		msg    *admintypes.MsgSubmitProposal
		sdkMsg sdk.Msg
	)

	cdc := m.AdminKeeper.Codec()
	err := cdc.UnmarshalInterfaceJSON([]byte(proposal.ProposalExecuteMessage.Message), &sdkMsg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall incoming sdk message")
	}

	signers, _, err := cdc.GetMsgV1Signers(sdkMsg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signers from incoming sdk message")
	}
	if len(signers) != 1 {
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "should be 1 signer")
	}
	if !sdk.AccAddress(signers[0]).Equals(authority) {
		return nil, errors.Wrap(sdkerrors.ErrUnauthorized, "authority in incoming msg is not equal to admin module")
	}

	msg, err = admintypes.NewMsgSubmitProposal([]sdk.Msg{sdkMsg}, contractAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create MsgSubmitProposal ")
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, errors.Wrap(err, "failed to validate incoming SubmitAdminProposal message")
	}

	response, err := m.Adminserver.SubmitProposal(ctx, msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to submit proposal")
	}

	return response, nil
}

// removed: createDenom helper

// PerformCreateDenom is used with createDenom to create a token denom; validates the msgCreateDenom.
// removed: PerformCreateDenom

// createDenom forces a transfer of a tokenFactory token
// removed: forceTransfer helper

// removed: PerformForceTransfer

// removed: setDenomMetadata helper

// removed: PerformSetDenomMetadata

// mintTokens mints tokens of a specified denom to an address.
// removed: mintTokens helper

// setBeforeSendHook sets before send hook for a specified denom.
// removed: setBeforeSendHook helper

// PerformMint used with mintTokens to validate the mint message and mint through token factory.
// removed: PerformMint

// removed: PerformSetBeforeSendHook

// changeAdmin changes the admin.
// removed: changeAdmin helper

// ChangeAdmin is used with changeAdmin to validate changeAdmin messages and to dispatch.
// removed: ChangeAdmin

// burnTokens burns tokens.
// removed: burnTokens helper

// PerformBurn performs token burning after validating tokenBurn message.
// removed: PerformBurn

// GetFullDenom is a function, not method, so the message_plugin can use it
// removed: GetFullDenom

// parseAddress parses address from bech32 string and verifies its format.
func parseAddress(addr string) (sdk.AccAddress, error) {
	parsed, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return nil, errors.Wrap(err, "address from bech32")
	}

	err = sdk.VerifyAddressFormat(parsed)
	if err != nil {
		return nil, errors.Wrap(err, "verify address format")
	}

	return parsed, nil
}

func (m *CustomMessenger) performSubmitTx(ctx sdk.Context, contractAddr sdk.AccAddress, submitTx *bindings.SubmitTx) (*ictxtypes.MsgSubmitTxResponse, error) {
	tx := ictxtypes.MsgSubmitTx{
		FromAddress:         contractAddr.String(),
		ConnectionId:        submitTx.ConnectionId,
		Memo:                submitTx.Memo,
		InterchainAccountId: submitTx.InterchainAccountId,
		Timeout:             submitTx.Timeout,
		Fee:                 submitTx.Fee,
	}
	for _, msg := range submitTx.Msgs {
		tx.Msgs = append(tx.Msgs, &types.Any{
			TypeUrl: msg.TypeURL,
			Value:   msg.Value,
		})
	}

	response, err := m.Ictxmsgserver.SubmitTx(ctx, &tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to submit interchain transaction")
	}

	return response, nil
}

func (m *CustomMessenger) registerInterchainAccount(ctx sdk.Context, contractAddr sdk.AccAddress, reg *bindings.RegisterInterchainAccount) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	response, err := m.performRegisterInterchainAccount(ctx, contractAddr, reg)
	if err != nil {
		ctx.Logger().Debug("performRegisterInterchainAccount: failed to register interchain account",
			"from_address", contractAddr.String(),
			"connection_id", reg.ConnectionId,
			"interchain_account_id", reg.InterchainAccountId,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to register interchain account")
	}

	data, err := json.Marshal(response)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal register interchain account response to JSON",
			"from_address", contractAddr.String(),
			"connection_id", reg.ConnectionId,
			"interchain_account_id", reg.InterchainAccountId,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	ctx.Logger().Debug("registered interchain account",
		"from_address", contractAddr.String(),
		"connection_id", reg.ConnectionId,
		"interchain_account_id", reg.InterchainAccountId,
	)

	anyResp, err := types.NewAnyWithValue(response)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", response)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) performRegisterInterchainAccount(ctx sdk.Context, contractAddr sdk.AccAddress, reg *bindings.RegisterInterchainAccount) (*ictxtypes.MsgRegisterInterchainAccountResponse, error) {
	// parse incoming ordering. If nothing passed, use ORDERED by default
	var orderValue channeltypes.Order
	if reg.Ordering == "" {
		orderValue = channeltypes.ORDERED
	} else {
		orderValueInt, ok := channeltypes.Order_value[reg.Ordering]

		if !ok {
			return nil, fmt.Errorf("failed to register interchain account: incorrect order value passed: %s", reg.Ordering)
		}
		orderValue = channeltypes.Order(orderValueInt)
	}

	msg := ictxtypes.MsgRegisterInterchainAccount{
		FromAddress:         contractAddr.String(),
		ConnectionId:        reg.ConnectionId,
		InterchainAccountId: reg.InterchainAccountId,
		RegisterFee:         getRegisterFee(reg.RegisterFee),
		Ordering:            orderValue,
	}

	response, err := m.Ictxmsgserver.RegisterInterchainAccount(ctx, &msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to register interchain account")
	}

	return response, nil
}

func (m *CustomMessenger) registerInterchainQuery(ctx sdk.Context, contractAddr sdk.AccAddress, reg *bindings.RegisterInterchainQuery) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	response, err := m.performRegisterInterchainQuery(ctx, contractAddr, reg)
	if err != nil {
		ctx.Logger().Debug("performRegisterInterchainQuery: failed to register interchain query",
			"from_address", contractAddr.String(),
			"query_type", reg.QueryType,
			"kv_keys", icqtypes.KVKeys(reg.Keys).String(),
			"transactions_filter", reg.TransactionsFilter,
			"connection_id", reg.ConnectionId,
			"update_period", reg.UpdatePeriod,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to register interchain query")
	}

	data, err := json.Marshal(response)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal register interchain query response to JSON",
			"from_address", contractAddr.String(),
			"kv_keys", icqtypes.KVKeys(reg.Keys).String(),
			"transactions_filter", reg.TransactionsFilter,
			"connection_id", reg.ConnectionId,
			"update_period", reg.UpdatePeriod,
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	ctx.Logger().Debug("registered interchain query",
		"from_address", contractAddr.String(),
		"query_type", reg.QueryType,
		"kv_keys", icqtypes.KVKeys(reg.Keys).String(),
		"transactions_filter", reg.TransactionsFilter,
		"connection_id", reg.ConnectionId,
		"update_period", reg.UpdatePeriod,
		"query_id", response.Id,
	)

	anyResp, err := types.NewAnyWithValue(response)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", response)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) performRegisterInterchainQuery(ctx sdk.Context, contractAddr sdk.AccAddress, reg *bindings.RegisterInterchainQuery) (*icqtypes.MsgRegisterInterchainQueryResponse, error) {
	msg := icqtypes.MsgRegisterInterchainQuery{
		Keys:               reg.Keys,
		TransactionsFilter: reg.TransactionsFilter,
		QueryType:          reg.QueryType,
		ConnectionId:       reg.ConnectionId,
		UpdatePeriod:       reg.UpdatePeriod,
		Sender:             contractAddr.String(),
	}

	response, err := m.Icqmsgserver.RegisterInterchainQuery(ctx, &msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to register interchain query")
	}

	return response, nil
}

func (m *CustomMessenger) validateProposalQty(proposal *bindings.AdminProposal) error {
	qty := 0
	if proposal.ParamChangeProposal != nil {
		qty++
	}
	if proposal.ProposalExecuteMessage != nil {
		qty++
	}

	switch qty {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("no admin proposal type is present in message")
	default:
		return fmt.Errorf("more than one admin proposal type is present in message")
	}
}

func (m *CustomMessenger) isLegacyProposal(proposal *bindings.AdminProposal) bool {
	switch {
	case proposal.ParamChangeProposal != nil:
		return true
	default:
		return false
	}
}

func (m *CustomMessenger) addSchedule(ctx sdk.Context, contractAddr sdk.AccAddress, addSchedule *bindings.AddSchedule) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	if !m.isAdmin(ctx, contractAddr) {
		return nil, nil, nil, errors.Wrap(sdkerrors.ErrUnauthorized, "only admin can add schedule")
	}

	authority := authtypes.NewModuleAddress(admintypes.ModuleName)

	msgs := make([]crontypes.MsgExecuteContract, 0, len(addSchedule.Msgs))
	for _, msg := range addSchedule.Msgs {
		msgs = append(msgs, crontypes.MsgExecuteContract{
			Contract: msg.Contract,
			Msg:      msg.Msg,
		})
	}

	_, err := m.CronMsgServer.AddSchedule(ctx, &crontypes.MsgAddSchedule{
		Authority:      authority.String(),
		Name:           addSchedule.Name,
		Period:         addSchedule.Period,
		Msgs:           msgs,
		ExecutionStage: crontypes.ExecutionStage(crontypes.ExecutionStage_value[addSchedule.ExecutionStage]),
	})
	if err != nil {
		ctx.Logger().Error("failed to addSchedule",
			"from_address", contractAddr.String(),
			"name", addSchedule.Name,
			"error", err,
		)
		return nil, nil, nil, errors.Wrapf(err, "failed to add %s schedule", addSchedule.Name)
	}

	ctx.Logger().Debug("schedule added",
		"from_address", contractAddr.String(),
		"name", addSchedule.Name,
		"period", addSchedule.Period,
	)

	return nil, nil, nil, nil
}

func (m *CustomMessenger) removeSchedule(ctx sdk.Context, contractAddr sdk.AccAddress, removeSchedule *bindings.RemoveSchedule) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	params, err := m.CronQueryServer.Params(ctx, &crontypes.QueryParamsRequest{})
	if err != nil {
		ctx.Logger().Error("failed to removeSchedule", "error", err)
		return nil, nil, nil, errors.Wrap(err, "failed to removeSchedule")
	}

	if !m.isAdmin(ctx, contractAddr) && contractAddr.String() != params.Params.SecurityAddress {
		return nil, nil, nil, errors.Wrap(sdkerrors.ErrUnauthorized, "only admin or security dao can remove schedule")
	}

	authority := authtypes.NewModuleAddress(admintypes.ModuleName)

	_, err = m.CronMsgServer.RemoveSchedule(ctx, &crontypes.MsgRemoveSchedule{
		Authority: authority.String(),
		Name:      removeSchedule.Name,
	})
	if err != nil {
		ctx.Logger().Error("failed to removeSchedule",
			"from_address", contractAddr.String(),
			"name", removeSchedule.Name,
			"error", err,
		)
		return nil, nil, nil, errors.Wrapf(err, "failed to remove %s schedule", removeSchedule.Name)
	}

	ctx.Logger().Debug("schedule removed",
		"from_address", contractAddr.String(),
		"name", removeSchedule.Name,
	)
	return nil, nil, nil, nil
}

func (m *CustomMessenger) resubmitFailure(ctx sdk.Context, contractAddr sdk.AccAddress, resubmitFailure *bindings.ResubmitFailure) ([]sdk.Event, [][]byte, [][]*types.Any, error) {
	failure, err := m.ContractmanagerQueryServer.AddressFailure(ctx, &contractmanagertypes.QueryFailureRequest{
		Address:   contractAddr.String(),
		FailureId: resubmitFailure.FailureId,
	})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "no failure with given FailureId found to resubmit")
	}

	_, err = m.ContractmanagerMsgServer.ResubmitFailure(ctx, &contractmanagertypes.MsgResubmitFailure{
		Sender:    contractAddr.String(),
		FailureId: resubmitFailure.FailureId,
	})
	if err != nil {
		ctx.Logger().Error("failed to resubmitFailure",
			"from_address", contractAddr.String(),
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "failed to resubmitFailure")
	}

	resp := bindings.ResubmitFailureResponse{FailureId: resubmitFailure.FailureId}
	data, err := json.Marshal(&resp)
	if err != nil {
		ctx.Logger().Error("json.Marshal: failed to marshal remove resubmitFailure response to JSON",
			"from_address", contractAddr.String(),
			"error", err,
		)
		return nil, nil, nil, errors.Wrap(err, "marshal json failed")
	}

	// Return failure for reverse compatibility purposes.
	// Maybe it'll be removed in the future because it was already deleted after resubmit before returning here.
	anyResp, err := types.NewAnyWithValue(failure)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to convert {%T} to Any", failure)
	}
	msgResponses := [][]*types.Any{{anyResp}}
	return nil, [][]byte{data}, msgResponses, nil
}

func (m *CustomMessenger) isAdmin(ctx sdk.Context, contractAddr sdk.AccAddress) bool {
	for _, admin := range m.AdminKeeper.GetAdmins(ctx) {
		if admin == contractAddr.String() {
			return true
		}
	}

	return false
}

func getRegisterFee(fee sdk.Coins) sdk.Coins {
	if fee == nil {
		return make(sdk.Coins, 0)
	}
	return fee
}
