package client

import (
	"github.com/neutron-org/neutron/v5/x/pool-incentives/client/cli"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	UpdatePoolIncentivesHandler  = govclient.NewProposalHandler(cli.NewCmdSubmitUpdatePoolIncentivesProposal)
	ReplacePoolIncentivesHandler = govclient.NewProposalHandler(cli.NewCmdSubmitReplacePoolIncentivesProposal)
)
