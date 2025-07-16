package client

import (
	"github.com/neutron-org/neutron/v5/x/incentives/client/cli"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	HandleCreateGroupsProposal = govclient.NewProposalHandler(cli.NewCmdHandleCreateGroupsProposal)
)
