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

    mintburn "github.com/maany-xyz/maany-dex/v5/x/mintburn/keeper"
    "github.com/maany-xyz/maany-dex/v5/x/mintburn/types"
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

// ---------------------
// Channel lifecycle
// ---------------------

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

func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

func contains(list []string, x string) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}

func (im IBCMiddleware) HandleChannelIdStorage(
    ctx sdk.Context,
    portID string,
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
    // Security binding: local connection's local client ID must equal the CCV provider client ID
    localClientID := connection.ClientId
    providerClientID, ok := im.keeper.ConsumerKeeper.GetProviderClientID(ctx)
    if !ok || providerClientID == "" {
        ctx.Logger().Info("mintburn: cannot fetch CCV provider client id; not allowlisting channel",
            "channel_id", channelID,
        )
        return nil
    }
    if localClientID != providerClientID {
        ctx.Logger().Info("mintburn: channel client mismatch; not allowlisting",
            "channel_id", channelID,
            "connection_id", connectionID,
            "local_client_id", localClientID,
            "provider_client_id", providerClientID,
        )
        return nil
    }

    // Passed security checks: allowlist the local transfer channel for mintburn path
    store := prefix.NewStore(ctx.KVStore(im.keeper.StoreKey), types.KeyAllowedChan)
    if isOpening {
        store.Set([]byte(channelID), []byte{1})
        ctx.Logger().Info("mintburn: set allowed channel (via CCV client bind)", "channel_id", channelID)
    } else {
        store.Delete([]byte(channelID))
        ctx.Logger().Info("mintburn: removed allowed channel", "channel_id", channelID)
    }
    return nil
}

func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID string,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, true); err != nil {
		return err
	}
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID string,
	channelID string,
) error {
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, true); err != nil {
		return err
	}
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID string,
	channelID string,
) error {
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, false); err != nil {
		return err
	}
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID string,
	channelID string,
) error {
	if err := im.HandleChannelIdStorage(ctx, portID, channelID, false); err != nil {
		return err
	}
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// ---------------------
// Packet lifecycle
// ---------------------

func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// proofID is a unique key for a packet on the receiving chain.
func proofID(pkt channeltypes.Packet) []byte {
	// DestinationPort/Channel because we're on the receiver
	return []byte(fmt.Sprintf("%s/%s/%d", pkt.DestinationPort, pkt.DestinationChannel, pkt.Sequence))
}

func (im IBCMiddleware) OnRecvPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer sdk.AccAddress,
) exported.Acknowledgement {
	okAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	// Parse ICS-20 payload
	var data ibctransfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		ctx.Logger().Error("mintburn: bad packet data", "err", err)
		return channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid packet data"))
	}

    // Security: verify that the transfer channel is bound to the CCV provider client
    // 1) Verify port/channels
    localAllowed := im.keeper.IsAllowedChannel(ctx, packet.DestinationChannel)
    if packet.DestinationPort != "transfer" {
        ctx.Logger().Info("mintburn: non-transfer destination port; forwarding",
            "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel,
            "src_port", packet.SourcePort, "src_channel", packet.SourceChannel,
        )
        return im.app.OnRecvPacket(ctx, packet, relayer)
    }
    if !localAllowed {
        ctx.Logger().Info("mintburn: channel not allowlisted; forwarding",
            "dst_channel", packet.DestinationChannel,
            "src_channel", packet.SourceChannel,
        )
        return im.app.OnRecvPacket(ctx, packet, relayer)
    }

    // 2) Resolve and verify the local connection's client against CCV provider client
    ch, found := im.keeper.ChannelKeeper.GetChannel(ctx, packet.DestinationPort, packet.DestinationChannel)
    if !found {
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("mintburn: channel not found"))
    }
    connID := ch.ConnectionHops[0]
    conn, found := im.keeper.ConnectionKeeper.GetConnection(ctx, connID)
    if !found {
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("mintburn: connection not found"))
    }
    localClientID := conn.ClientId
    providerClientID, ok := im.keeper.ConsumerKeeper.GetProviderClientID(ctx)
    if !ok || providerClientID == "" || localClientID != providerClientID {
        ctx.Logger().Info("mintburn: unauthorized transfer path; rejecting",
            "dst_channel", packet.DestinationChannel,
            "local_client_id", localClientID,
            "provider_client_id", providerClientID,
        )
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("unauthorized transfer path"))
    }

    params := im.keeper.GetParams(ctx)
    if params.Pause {
        ctx.Logger().Info("mintburn: paused; rejecting packet")
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("mintburn paused"))
    }


	// Resolve base denom (trace-aware)
	baseDenom := data.Denom
	if !ibctransfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {
		trace := ibctransfertypes.ParseDenomTrace(data.Denom)
		baseDenom = trace.BaseDenom
	}
    if !contains(params.AllowedBaseDenoms, baseDenom) {
        // Not our special path -> pass to ICS-20 app (will mint voucher)
        ctx.Logger().Info("mintburn: base denom not allowed; forwarding",
            "base_denom", baseDenom, "allowed", fmt.Sprintf("%v", params.AllowedBaseDenoms),
        )
        return im.app.OnRecvPacket(ctx, packet, relayer)
    }

	// Replay guard
	pid := proofID(packet)
    if im.keeper.HasProof(ctx, pid) {
        ctx.Logger().Error("mintburn: duplicate proof", "proof", string(pid))
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("duplicate proof"))
    }

	// Validate receiver and amount
    rcpt, err := sdk.AccAddressFromBech32(data.Receiver)
    if err != nil {
        ctx.Logger().Info("mintburn: invalid receiver address", "receiver", data.Receiver)
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid receiver address"))
    }
    amt, ok := sdkmath.NewIntFromString(data.Amount)
    if !ok || !amt.IsPositive() {
        ctx.Logger().Info("mintburn: invalid amount", "amount", data.Amount)
        return channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid token amount"))
    }

	// Mint mirrored native on DEX (use your DEX minimal denom here)
	mintCoin := sdk.NewCoin(params.DexNativeDenom, amt) // or "umaany" if that's your DEX denom
	if err := im.keeper.MintTokens(ctx, rcpt, mintCoin); err != nil {
		ctx.Logger().Error("mintburn: mint failed", "err", err)
		return channeltypes.NewErrorAcknowledgement(fmt.Errorf("mint failed"))
	}

	// Mark proof consumed and bump mirror supply
	im.keeper.ConsumeProof(ctx, pid)
	im.keeper.AddMirrorSupply(ctx, amt)

	ctx.Logger().Info("mintburn: minted mirrored on DEX",
		"amount", mintCoin.String(), "receiver", rcpt.String(), "proof", string(pid))

	// IMPORTANT: do NOT forward to transfer app for handled path (prevents voucher mint)
	return okAck
}

// DEX middleware: OnAcknowledgementPacket (only for DEX->Provider sends)
func (im IBCMiddleware) OnAcknowledgementPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    acknowledgement []byte,
    relayer sdk.AccAddress,
) error {
    // Parse ack
    var ack channeltypes.Acknowledgement
    if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Only act on success
    if _, ok := ack.Response.(*channeltypes.Acknowledgement_Result); !ok {
        // error/timeouts -> ICS-20 handles refund; we do nothing
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Decode packet data
    var data ibctransfertypes.FungibleTokenPacketData
    if err := json.Unmarshal(packet.GetData(), &data); err != nil {
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Only for our allow-listed channel
    if packet.SourcePort != "transfer" || !im.keeper.IsAllowedChannel(ctx, packet.SourceChannel) {
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Resolve base denom (trace-aware) and verify it is our DEX native (e.g., "umaany")
    baseDenom := data.Denom
    if !ibctransfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {
        trace := ibctransfertypes.ParseDenomTrace(data.Denom)
        baseDenom = trace.BaseDenom
    }
	//TODO:
    // If you have params: compare with params.DexNativeDenom instead of hardcoding
    if baseDenom != "umaany" { // or "stake" depending on your DEX denom
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Parse amount
    amt, ok := sdkmath.NewIntFromString(data.Amount)
    if !ok || !amt.IsPositive() {
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Burn tokens from the DEX escrow to remove mirrored supply
    escrowAddr := ibctransfertypes.GetEscrowAddress(packet.SourcePort, packet.SourceChannel)
    coin := sdk.NewCoin(baseDenom, amt)
    if err := im.keeper.BurnEscrowedTokens(ctx, escrowAddr, coin); err != nil {
        // If burn fails, don't block the core app ack flow; just log and continue
        ctx.Logger().Error("mintburn: burn escrow on DEX failed", "err", err)
        return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }

    // Decrement mirror supply
    im.keeper.SubMirrorSupply(ctx, amt)

    ctx.Logger().Info("mintburn: burned mirrored on DEX after ACK",
        "amount", coin.String(), "escrow", escrowAddr.String())

    return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}
