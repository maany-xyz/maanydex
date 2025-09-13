package types

const (
	ModuleName   = "genesismint"
	StoreKey     = "x-genesismint"
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
	ClaimedPrefix = []byte{0x11} // claimed escrow ids
)

// key: claimed/<provider_chain_id>/<escrow_id> -> []byte{1}
func ClaimedKey(providerChainID, escrowID string) []byte {
	// simple: claimed|provider|escrow
	k := append([]byte("claimed|"), []byte(providerChainID)...)
	k = append(k, []byte("|")...)
	k = append(k, []byte(escrowID)...)
	return append(ClaimedPrefix, k...)
}
