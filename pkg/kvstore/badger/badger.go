package badger

import (
	"log"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/kvanticoss/gdpr-id-mapper/pkg/kvstore"
)

var _ kvstore.Store = (*Adapter)(nil)

// Adapter adapter ensures that the badger key/values store implmentes our interface
type Adapter struct {
	db *badger.DB
}

// NewAdapter returns a new Adapter for key/value logic
func NewAdapter(db *badger.DB) *Adapter {
	return &Adapter{db}
}

// Close ensures the db is flushed to disk and put into a good state
func (b *Adapter) Close() {
	b.db.Close()
}

// Put store a []byte value at a given []byte key
func (b *Adapter) Put(key []byte, value []byte, ttl *time.Duration) error {
	return b.db.Update(func(tx *badger.Txn) error {
		if ttl != nil {
			e := badger.NewEntry(key, value).WithTTL(*ttl)
			return tx.SetEntry(e)
		}
		return tx.Set(key, value)
	})
}

// PrefixScan iterates all over all []byte keys where bytes.HasPrefix(key, prefix) == true; and calls the callback with the keys and values.
func (b *Adapter) PrefixScan(
	prefix []byte,
	mapper func(key []byte, value []byte) error,
) error {
	return b.db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			if err := item.Value(func(v []byte) error {
				return mapper(k, v)
			}); err != nil {
				return err
			}

		}
		return nil
	})
}

// Get returns the []byte value of a specific key; or nil otherwise
func (b *Adapter) Get(key []byte) []byte {
	value := []byte{}

	err := b.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(value)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil && err != badger.ErrKeyNotFound {
		log.Printf("Error getting key:%x %v", key, err)
	}
	return value
}

// Delete the record at the []byte value. Returns 1,nil on success or 0, error on failure A call to get directly afterwards should return nil
func (b *Adapter) Delete(key []byte) (int, error) {
	err := b.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(key)
	})
	if err == nil {
		return 1, nil
	}
	return 0, err
}

// DeletePrefix should delete all records for which bytes.HasPrefix(key, prefix) is true; Returns the number of records deleted.
func (b *Adapter) DeletePrefix(prefix []byte) (int, error) {
	deleted := 0
	err := b.db.Update(func(tx *badger.Txn) error {

		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			if err := tx.Delete(item.Key()); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	if err == nil {
		return deleted, nil
	}
	return deleted, err
}
