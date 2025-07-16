package client

import (
	"github.com/neutron-org/neutron/v5/x/concentrated-liquidity/client/cli"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	TickSpacingDecreaseProposalHandler = govclient.NewProposalHandler(cli.NewTickSpacingDecreaseProposal)
)
