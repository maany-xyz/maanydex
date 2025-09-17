package types

import "fmt"

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
    DonePrefix = []byte{0x14} // module completion flag
    ConfigPrefix = []byte{0x15} // module configuration
    ICAPacketPrefix = []byte{0x16} // mapping from (channel,seq) -> escrow id
    InflightPrefix = []byte{0x17} // in-flight claims awaiting ack
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

// key: done -> []byte{1}
func DoneKey() []byte {
    return append(DonePrefix, []byte("done")...)
}

// ---- Config keys ----
// key: cfg|ica_connection_id -> string
func ConfigICAConnectionIDKey() []byte { return append(ConfigPrefix, []byte("cfg|ica_connection_id")...) }
// key: cfg|ica_owner -> string
func ConfigICAOwnerKey() []byte { return append(ConfigPrefix, []byte("cfg|ica_owner")...) }
// key: cfg|ica_timeout_seconds -> uint64 (ASCII)
func ConfigICATimeoutSecondsKey() []byte { return append(ConfigPrefix, []byte("cfg|ica_timeout_seconds")...) }
// key: cfg|max_claims_per_block -> uint64 (ASCII)
func ConfigMaxClaimsPerBlockKey() []byte { return append(ConfigPrefix, []byte("cfg|max_claims_per_block")...) }

// key: pkt|<channel_id>|<sequence> -> escrow_id
func ICAPacketKey(channelID string, sequence uint64) []byte {
    s := fmt.Sprintf("pkt|%s|%d", channelID, sequence)
    return append(ICAPacketPrefix, []byte(s)...)
}

// key: inflight|<provider_chain_id>|<escrow_id> -> 1
func InflightKey(providerChainID, escrowID string) []byte {
    k := append([]byte("inflight|"), []byte(providerChainID)...)
    k = append(k, '|')
    k = append(k, []byte(escrowID)...)
    return append(InflightPrefix, k...)
}
