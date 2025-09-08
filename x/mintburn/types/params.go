package types

import (
	"fmt"
	"regexp"
	"strings"
)

type Params struct {
	// Optional: who can update params later (gov module addr or multisig)
	Authority string `json:"authority" yaml:"authority"`

	// Emergency kill-switch for minting on DEX receive
	Pause bool `json:"pause" yaml:"pause"`

	// Base denoms accepted from Provider on Provider→DEX (e.g., ["stake"])
	AllowedBaseDenoms []string `json:"allowed_base_denoms" yaml:"allowed_base_denoms"`

	// Counterparty chain-ids allowed during channel open (Provider side for this direction)
	ProviderChainIDs []string `json:"provider_chain_ids" yaml:"provider_chain_ids"`

	// Denom to mint on DEX when receiving from Provider (e.g., "umaany")
	DexNativeDenom string `json:"dex_native_denom" yaml:"dex_native_denom"`
}

func DefaultParams() Params {
	return Params{
		Authority:         "",
		Pause:             false,
		AllowedBaseDenoms: []string{"stake"},
		ProviderChainIDs:  []string{"maany-mainnet"},
		DexNativeDenom:    "umaany",
	}
}

func (p Params) Validate() error {
	reCID := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-]{2,64}$`)
	if len(p.ProviderChainIDs) == 0 {
		return fmt.Errorf("provider_chain_ids must not be empty")
	}
	for _, cid := range p.ProviderChainIDs {
		if !reCID.MatchString(cid) {
			return fmt.Errorf("invalid provider chain-id: %q", cid)
		}
	}
	if len(p.AllowedBaseDenoms) == 0 {
		return fmt.Errorf("allowed_base_denoms must not be empty")
	}
	for _, d := range p.AllowedBaseDenoms {
		if strings.TrimSpace(d) == "" {
			return fmt.Errorf("empty denom in allowed_base_denoms")
		}
	}
	if strings.TrimSpace(p.DexNativeDenom) == "" {
		return fmt.Errorf("dex_native_denom must not be empty")
	}
	return nil
}
