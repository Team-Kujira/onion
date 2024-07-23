package types

const (
	ModuleName = "onion"
	RouterKey  = ModuleName
	StoreKey   = ModuleName

	OnionSequencePrefix = "onion-sequence"
)

var (
	ParamsKey = []byte("p_onion")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const AccountNumber = ^uint64(0)
