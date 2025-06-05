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

//var _ porttypes.Middleware = &IBCMiddleware{}

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

func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	_ sdk.AccAddress,
) error {
	//var ack channeltypes.Acknowledgement
	
	return nil
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanCloseInit callback.
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	ctx.Logger().Info("In OnChanOpenAck!")
	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	if portID == "transfer" {
		ctx.Logger().Info("In OnChanOpenAck: registering a new tranfer channel.")
		channel, found := im.keeper.ChannelKeeper.GetChannel(ctx, portID, channelID)
		if !found {
			return fmt.Errorf("channel not found")
		}
		ctx.Logger().Info("Channel Found")
		connectionID := channel.ConnectionHops[0]
		ctx.Logger().Info("Connection Id: ", "ID", connectionID)
		connection, found := im.keeper.ConnectionKeeper.GetConnection(ctx, connectionID)
		if !found {
			return fmt.Errorf("connection %s not found", connectionID)
		}
		ctx.Logger().Info("Connection found.")
		clientID := connection.Counterparty.ClientId
		ctx.Logger().Info("Cliend ID: ", "ID", clientID)
		clientState, found := im.keeper.ClientKeeper.GetClientState(ctx, clientID)
		if !found {
			return fmt.Errorf("client state for %s not found", clientID)
		}
		ctx.Logger().Info("Client state found.")
		tmClientState, ok := clientState.(*ibctmtypes.ClientState)
		if !ok {
			return fmt.Errorf("unexpected client state type")
		}
		ctx.Logger().Info("TM client state found.", "State", tmClientState.String())
		if tmClientState.ChainId == "maany-mainnet" {
			store := prefix.NewStore(ctx.KVStore(im.keeper.StoreKey), []byte("allowed-channel/"))
			store.Set([]byte(channelID), []byte{1})
			ctx.Logger().Info("Successfully set channel-id", "ID", channelID)
		}

	}

	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}
 
// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanOpenConfirm callback.
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
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
	// call underlying app's OnChanOpenInit callback with the appVersion
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

// // Derive the IBC-prefixed denom
// func deriveIBCDenom(packet channeltypes.Packet, originalDenom string) string {
//     sourcePort := packet.SourcePort
//     sourceChannel := packet.SourceChannel
// 	prefixDenom := ibctransfertypes.GetPrefixedDenom(sourcePort, sourceChannel, originalDenom)
// 	hash := sha256.Sum256([]byte(prefixDenom))
//     return "ibc/" + strings.ToUpper(hex.EncodeToString(hash[:]))
// }

func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {
	// executes the IBC transfer OnRecv logic
	ctx.Logger().Info("In OnRecvPacket mintburn")

	//ack := im.app.OnRecvPacket(ctx, packet, relayer)
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	// Note that inside the below if condition statement,
	// we know that the IBC transfer succeeded. That entails
	// that the packet data is valid and can be safely
	// deserialized without checking errors.
	if ack.Success() {
		// execute the middleware logic only if the sender is a consumer chain


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

		coinDenom := data.Denom //still "stake" at this point
        coinAmt, ok := sdkmath.NewIntFromString(data.Amount)
        if !ok {
            ctx.Logger().Error("invalid token amount", "amount", data.Amount)
			ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid token amount"))
            return ack
        }

		// ibcDenom := deriveIBCDenom(packet, data.Denom)
		// bridgedDenom := "ibc/3C3D7B3BE4ECC85A0E5B52A3AEC3B7DFC2AA9CA47C37821E57020D6807043BE9"
		// ctx.Logger().Info("Hex values","ibcDenom", fmt.Sprintf("%x", ibcDenom), "bridgedDenom", fmt.Sprintf("%x", bridgedDenom))
		if packet.SourcePort == "transfer" && im.keeper.IsAllowedChannel(ctx, packet.SourceChannel) && data.Denom == "stake" {
			ctx.Logger().Info("The denom for exchange is correct")
			//NOTE: this should be the native token of the dex consumer
			nativeToken := sdk.NewCoin("stake", coinAmt)
			// bridgedToken := sdk.NewCoin(ibcDenom, coinAmt)

			// ctx.Logger().Info("the native token is: ", "nat", nativeToken)
			// ctx.Logger().Info("the native bridged token is: ", "br", bridgedToken)

			 // Mint native tokens
            if err := im.keeper.MintTokens(ctx, receiver, nativeToken); err != nil {
                ctx.Logger().Error("failed to mint tokens", "error", err)
				ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed to mint tokens"))
                return ack
            }
			ctx.Logger().Info("Successfully minted native tokens", "amount", nativeToken)
		} else {
			ctx.Logger().Info("This is a regular ibc transfer of: ", "denom",  coinDenom)
			ctx.Logger().Info("Passing forward to core ibc module: ")
			ack := im.app.OnRecvPacket(ctx, packet, relayer)
			return ack

		}
	} else {
		ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed acknowledgement"))
		return ack
	}

	return ack
}