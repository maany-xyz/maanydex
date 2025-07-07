package mintburn

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	mintburn "github.com/neutron-org/neutron/v5/x/mintburn/keeper"

	ibctmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

type IBCMiddleware struct {
	app    porttypes.IBCModule
	keeper mintburn.Keeper
}

func NewIBCMiddleware(app porttypes.IBCModule, k mintburn.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:    app,
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	// call underlying app's OnChanOpenTry callback with the appVersion
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

func (im IBCMiddleware) HandleChannelIdStorage(
	ctx sdk.Context,
	portID,
	channelID string,
	isOpening bool,
) error {
	if portID != "transfer" {
		return nil
	}

	channel, found := im.keeper.ChannelKeeper.GetChannel(ctx, portID, channelID)
	if !found {
		return fmt.Errorf("channel not found")
	}
	connectionID := channel.ConnectionHops[0]
	connection, found := im.keeper.ConnectionKeeper.GetConnection(ctx, connectionID)
	if !found {
		return fmt.Errorf("connection %s not found", connectionID)
	}
	clientID := connection.Counterparty.ClientId
	clientState, found := im.keeper.ClientKeeper.GetClientState(ctx, clientID)
	if !found {
		return fmt.Errorf("client state for %s not found", clientID)
	}
	tmClientState, ok := clientState.(*ibctmtypes.ClientState)
	if !ok {
		return fmt.Errorf("unexpected client state type")
	}
	if tmClientState.ChainId == "maany-mainnet" {

		if isOpening {
			store := prefix.NewStore(ctx.KVStore(im.keeper.StoreKey), []byte("allowed-channel/"))
			store.Set([]byte(channelID), []byte{1})
			ctx.Logger().Info("Successfully set channel-id", "ID", channelID)
		} else {
			store := prefix.NewStore(ctx.KVStore(im.keeper.StoreKey), []byte("allowed-channel/"))
			store.Delete([]byte(channelID))
			ctx.Logger().Info("Channel deleted in OnChanCloseConfirm", "ID", channelID)
		}
		
	}

	return nil
}

func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID,
	counterpartyChannelID,
	counterpartyVersion string,
) error {
	// Run middleware logic first
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, true); err != nil {
		//On any error concerning establishing a transfer channel, return the error and aboart the channel creation 
		return err
	}
	// Then forward to the underlying IBC app
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}
 
// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	//Note: since a channel can be opened from both ways, need to handle same logic here as for OnChanOpenAck
	// Run middleware logic first
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, true); err != nil {
		//On any error concerning establishing a transfer channel, return the error and aboart the channel creation 
		return err
	}
	// Then forward to the underlying IBC app
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Run middleware logic first
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, false); err != nil {
	//On any error concerning establishing a transfer channel, return the error and aboart the channel closure 
		return err
	}
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Run middleware logic first
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, false); err != nil {
	//On any error concerning establishing a transfer channel, return the error and aboart the channel closure 
		return err
	}
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnTimeoutPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// call underlying app's OnTimeoutPacket callback.
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	var data ibctransfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		ctx.Logger().Error("failed to unmarshal packet data", "error", err)
		return ack
	}

	receiver, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		ctx.Logger().Error("invalid receiver address", "receiver", data.Receiver)
		ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid receiver address"))
		return ack
	}

	coinDenom := data.Denom
	coinAmt, ok := sdkmath.NewIntFromString(data.Amount)
	if !ok {
		ctx.Logger().Error("invalid token amount", "amount", data.Amount)
		ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid token amount"))
		return ack
	}

	if packet.SourcePort == "transfer" &&
		im.keeper.IsAllowedChannel(ctx, packet.SourceChannel) &&
		data.Denom == "stake" {

		ctx.Logger().Info("The denom for exchange is correct")
		nativeToken := sdk.NewCoin("stake", coinAmt)

		if err := im.keeper.MintTokens(ctx, receiver, nativeToken); err != nil {
			ctx.Logger().Error("failed to mint tokens", "error", err)
			ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed to mint tokens"))
			return ack
		}

		ctx.Logger().Info("Successfully minted native tokens", "amount", nativeToken)
	} else {
		ctx.Logger().Info("This is a regular IBC transfer", "denom", coinDenom)
		ctx.Logger().Info("Passing forward to core IBC module")
		ack := im.app.OnRecvPacket(ctx, packet, relayer)
		return ack
	}

	return ack
}

func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	
	var ack channeltypes.Acknowledgement
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		ctx.Logger().Error("Cant unmarshal ack", "err", err)
		return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	} else {

		if ack.Response == nil {
			ctx.Logger().Error("Acknowledgement response is nil")
			return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
		}

		switch resp := ack.Response.(type) {
			case *channeltypes.Acknowledgement_Error:
				// This was an error, handle refund logic or logging
				ctx.Logger().Info("Acknowledgement contains error", "error", resp.Error)
				return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)

			case *channeltypes.Acknowledgement_Result:
				ctx.Logger().Info("In case successful ack, initialte burning")
				var data ibctransfertypes.FungibleTokenPacketData
				if err := json.Unmarshal(packet.GetData(), &data); err != nil {
					ctx.Logger().Error("Cant unmarshal data", "err", err)
					return err
				}
				ctx.Logger().Info("unpacked data with: ","data",data)
				// Only burn if the denom is expected and the channel is whitelisted
				if packet.SourcePort == "transfer" &&
					im.keeper.IsAllowedChannel(ctx, packet.SourceChannel) &&
					data.Denom == "stake" {

					ctx.Logger().Info("is valid source port and channel-id")

					amount, ok := sdkmath.NewIntFromString(data.Amount)
					if !ok {
						ctx.Logger().Error("invalid token amount", "err", "")
						return fmt.Errorf("invalid token amount")
					}

					coin := sdk.NewCoin(data.Denom, amount)

					ctx.Logger().Info("In here with coin and amount", "amount", amount, "coin", coin)

					// Build escrow address and burn the amount
					escrowAddr := ibctransfertypes.GetEscrowAddress(packet.SourcePort, packet.SourceChannel)
					if err := im.keeper.BurnEscrowedTokens(ctx, escrowAddr, coin); err != nil {
						ctx.Logger().Error("Err burning tokens", "err", err)
						return nil
					}

					sdk.UnwrapSDKContext(ctx).Logger().Info("Successfully burned escrowed tokens after ACK",
						"coin", coin.String(), "escrow", escrowAddr.String())
				}
				
			default:
				ctx.Logger().Error("Unexpected acknowledgement type")
				return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
		}


	}

	return nil
}