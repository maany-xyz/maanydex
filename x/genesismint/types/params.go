package types

import "fmt"

// Attach validation to the generated Params type.
func (p Params) ValidateBasic() error {
	if p.ProviderClientId == "" || p.ProviderChainId == "" {
		return fmt.Errorf("provider client/chain id required")
	}
	if p.AllowedProviderDenom == "" || p.MintDenom == "" {
		return fmt.Errorf("denoms required")
	}
	return nil
}
