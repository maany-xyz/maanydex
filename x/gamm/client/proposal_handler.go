package client

import (
	"github.com/neutron-org/neutron/v5/x/gamm/client/cli"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	ReplaceMigrationRecordsProposalHandler    = govclient.NewProposalHandler(cli.NewCmdSubmitReplaceMigrationRecordsProposal)
	UpdateMigrationRecordsProposalHandler     = govclient.NewProposalHandler(cli.NewCmdSubmitUpdateMigrationRecordsProposal)
	CreateCLPoolAndLinkToCFMMProposalHandler  = govclient.NewProposalHandler(cli.NewCmdSubmitCreateCLPoolAndLinkToCFMMProposal)
	SetScalingFactorControllerProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitSetScalingFactorControllerProposal)
)
