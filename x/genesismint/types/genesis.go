package types

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           &Params{},       // pointer matches pb.go
		Mints:            []*MintIntent{}, // slice of pointers matches pb.go
		ClaimedEscrowIds: []string{},
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params missing")
	}
	if err := gs.Params.ValidateBasic(); err != nil {
		return fmt.Errorf("params: %w", err)
	}

	for i, m := range gs.Mints {
		if m == nil {
			return fmt.Errorf("mints[%d]: nil", i)
		}
		if m.ProviderChainId == "" {
			return fmt.Errorf("mints[%d]: provider_chain_id required", i)
		}
		if len(m.KeyPath) != 2 {
			return fmt.Errorf("mints[%d]: key_path must be [storeName, hexKey]", i)
		}
		if _, err := hex.DecodeString(m.KeyPath[1]); err != nil {
			return fmt.Errorf("mints[%d]: key_path[1] not valid hex: %w", i, err)
		}
		if m.Value == "" {
			return fmt.Errorf("mints[%d]: value (base64) required", i)
		}
		if _, err := base64.StdEncoding.DecodeString(m.Value); err != nil {
			return fmt.Errorf("mints[%d]: value not base64: %w", i, err)
		}
		if m.AmountDenom == "" || m.AmountValue == "" {
			return fmt.Errorf("mints[%d]: amount denom/value required", i)
		}
		if m.Recipient == "" {
			return fmt.Errorf("mints[%d]: recipient required", i)
		}
	}

	for i, id := range gs.ClaimedEscrowIds {
		if id == "" {
			return fmt.Errorf("claimed_escrow_ids[%d]: empty", i)
		}
	}
	return nil
}
