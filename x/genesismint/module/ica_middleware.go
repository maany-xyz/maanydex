package genesismint

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
    capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
    channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
    porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
    ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

    "github.com/maany-xyz/maany-dex/v5/x/genesismint/keeper"
)

// ICAMiddleware intercepts ICA acks to update genesismint state.
type ICAMiddleware struct {
    app    porttypes.IBCModule
    keeper keeper.Keeper
}

func NewICAMiddleware(app porttypes.IBCModule, k keeper.Keeper) ICAMiddleware {
    return ICAMiddleware{app: app, keeper: k}
}

// Delegate all channel lifecycle to next module
func (m ICAMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) (string, error) {
    return m.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}
func (m ICAMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
    return m.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}
func (m ICAMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID, counterpartyChannelID, counterpartyVersion string) error {
    return m.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}
func (m ICAMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
    return m.app.OnChanOpenConfirm(ctx, portID, channelID)
}
func (m ICAMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
    return m.app.OnChanCloseInit(ctx, portID, channelID)
}
func (m ICAMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
    return m.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// Packet lifecycle
func (m ICAMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
    return m.app.OnRecvPacket(ctx, packet, relayer)
}
func (m ICAMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
    // Clear inflight so the claim can be retried on next BeginBlock
    ctx.Logger().Error("genesismint: ICA packet timeout", "channel", packet.SourceChannel, "sequence", packet.Sequence)
    m.keeper.HandleICATimeout(ctx, packet)
    return m.app.OnTimeoutPacket(ctx, packet, relayer)
}
func (m ICAMiddleware) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
    // Parse ack via IBC channel codec
    var ack channeltypes.Acknowledgement
    if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
        ctx.Logger().Error("genesismint: failed to unmarshal ICA ack", "err", err)
        // proceed downstream anyway
        return m.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
    }
    switch ack.Response.(type) {
    case *channeltypes.Acknowledgement_Result:
        ctx.Logger().Info("genesismint: received ICA ack success", "channel", packet.SourceChannel, "sequence", packet.Sequence)
        m.keeper.HandleICAAckSuccess(ctx, packet)
    case *channeltypes.Acknowledgement_Error:
        ctx.Logger().Error("genesismint: received ICA ack error", "channel", packet.SourceChannel, "sequence", packet.Sequence)
        m.keeper.HandleICAAckError(ctx, packet)
    default:
        ctx.Logger().Error("genesismint: received ICA ack unknown type", "channel", packet.SourceChannel, "sequence", packet.Sequence)
    }
    return m.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}
