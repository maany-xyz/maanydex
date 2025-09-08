package types

type GenesisState struct {
	Params Params `json:"params" yaml:"params"`
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{Params: DefaultParams()}
}

func (gs *GenesisState) Validate() error {
	return gs.Params.Validate()
}
