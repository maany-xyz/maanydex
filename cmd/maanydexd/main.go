package main

import (
	"os"

	"github.com/maany-xyz/maany-dex/v5/app/config"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	//authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/maany-xyz/maany-dex/v5/app"
)

func main() {


	config := config.GetDefaultConfig()
	config.Seal()

	// Print Module account address:
	// moduleName := "mintburn" // The name of your module account
    // moduleAddress := authtypes.NewModuleAddress(moduleName)
    // fmt.Printf("Address for module '%s': %s\n", moduleName, moduleAddress.String())

	rootCmd, _ := NewRootCmd()

	rootCmd.AddCommand(AddConsumerSectionCmd(app.DefaultNodeHome))

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}


}
