package types

const (
	ModuleName            = "mintburn"
	StoreKey              = ModuleName
	MintBurnModuleAccount = ModuleName
)

// KV prefixes / keys used by the module.
var (
	// Whole Params blob (binary codec)
	KeyParams = []byte("params")

	// Replay guard for packets on DEX (packet-unique proof ids)
	KeyProofPrefix = []byte("proof/")

	// Allowlist of IBC transfer channel-ids set during channel open
	KeyAllowedChan = []byte("allowed-channel/")

	// String-encoded sdk.Int total of mirrored supply on DEX
	KeyMirrorSupply = []byte("mirror_supply")
)
