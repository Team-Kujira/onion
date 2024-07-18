package types

const (
	// ModuleName defines the module name
	ModuleName = "onion"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_onion"

    
)

var (
	ParamsKey = []byte("p_onion")
)



func KeyPrefix(p string) []byte {
    return []byte(p)
}
