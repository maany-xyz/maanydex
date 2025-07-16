package client

import (
	"github.com/neutron-org/neutron/v5/x/cosmwasmpool/client/cli"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	UploadCodeIdAndWhitelistProposalHandler = govclient.NewProposalHandler(cli.NewCmdUploadCodeIdAndWhitelistProposal)
	MigratePoolContractsProposalHandler     = govclient.NewProposalHandler(cli.NewCmdMigratePoolContractsProposal)
)
