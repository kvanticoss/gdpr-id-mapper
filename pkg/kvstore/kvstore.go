package kvstore

import "time"

// Store defines the baisc operations required by an KV implementation
type Store interface {
	// Put store a []byte value at a given []byte key
	Put(key []byte, value []byte, ttl *time.Duration) error

	// PrefixScan iterates all over all []byte keys where bytes.HasPrefix(key, prefix) == true; and calls the callback with the keys and values.
	//PrefixScan(prefix []byte, mapper func(key []byte, value []byte) error) error

	// return the []byte value of a specific key; or nil otherwise
	Get(key []byte) []byte

	// Delete the record at the []byte value. Returns 1,nil on success or 0, error on failure A call to get directly afterwards should return nil
	Delete(key []byte) (int, error)

	// DeletePrefix should delete all records for which bytes.HasPrefix(key, prefix) is true; Returns the number of records deleted.
	DeletePrefix(key []byte) (int, error)
}
