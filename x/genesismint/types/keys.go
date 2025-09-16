package types

const (
	ModuleName   = "genesismint"
	StoreKey     = "x-genesismint"
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
    ClaimedPrefix = []byte{0x11} // claimed escrow ids
    PendingClaimPrefix = []byte{0x12} // pending claims to be sent via ICA
    ICAPendingPrefix = []byte{0x13} // ICA registration pending flags
)

// key: claimed/<provider_chain_id>/<escrow_id> -> []byte{1}
func ClaimedKey(providerChainID, escrowID string) []byte {
	// simple: claimed|provider|escrow
	k := append([]byte("claimed|"), []byte(providerChainID)...)
	k = append(k, []byte("|")...)
	k = append(k, []byte(escrowID)...)
    return append(ClaimedPrefix, k...)
}

// key: pclaim/<provider_chain_id>/<escrow_id> -> []byte{1}
func PendingClaimKey(providerChainID, escrowID string) []byte {
    k := append([]byte("pclaim|"), []byte(providerChainID)...)
    k = append(k, []byte("|")...)
    k = append(k, []byte(escrowID)...)
    return append(PendingClaimPrefix, k...)
}

// key: ica|<connection_id>|<owner> -> []byte{1}
func ICAPendingKey(connectionID, owner string) []byte {
    k := append([]byte("ica|"), []byte(connectionID)...)
    k = append(k, []byte("|")...)
    k = append(k, []byte(owner)...)
    return append(ICAPendingPrefix, k...)
}
