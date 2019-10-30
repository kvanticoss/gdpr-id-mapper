package badger_test

import (
	"bytes"
	"testing"

	badger "github.com/dgraph-io/badger"
	badgerAdaptor "github.com/kvanticoss/gdpr-id-mapper/pkg/kvstore/badger"
	"github.com/stretchr/testify/assert"
)

func TestBoldDbAdaptor(t *testing.T) {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		t.Fatal(err.Error())
	}

	badger := badgerAdaptor.NewAdapter(db)
	defer badger.Close()

	//Simple read / Write
	key1 := []byte("testkey")
	val1 := []byte("testvalue")
	if err := badger.Put(key1, val1, nil); err != nil {
		t.Errorf("Error saving to key %x, %v", key1, err)
	}
	returnVal1 := badger.Get(key1)
	if !bytes.Equal(val1, returnVal1) {
		t.Errorf("Write and readback to same key doesn't yeild same value; put %s; got %s",
			string(val1),
			string(returnVal1))
	}

	//Remove said key
	_, err = badger.Delete(key1)
	assert.NoError(t, err)
	returnVal1 = badger.Get(key1)
	if bytes.Equal(val1, returnVal1) {
		t.Errorf("Write, Delete and readback yeilded same value, should be '' after delete got %v",
			string(returnVal1))
	}

	//Prefix Scan
	assert.NoError(t, badger.Put([]byte("key1"), []byte("val1"), nil))
	assert.NoError(t, badger.Put([]byte("key11"), []byte("val11"), nil))
	assert.NoError(t, badger.Put([]byte("key111"), []byte("val111"), nil))
	assert.NoError(t, badger.Put([]byte("key1111"), []byte("val1111"), nil))
	iteration := ""
	assert.NoError(t, badger.PrefixScan([]byte("key1"), func(key []byte, val []byte) error {
		iteration += "1"

		if !bytes.Equal(val, []byte("val"+iteration)) {
			t.Errorf("Prefix scan didn't yeild expected results; expected '%s', got '%s' ",
				"val"+iteration,
				string(val))
		}
		return nil
	}))

	//Prefix delete
	count, err := badger.DeletePrefix([]byte("key1"))
	if err != nil {
		t.Errorf("Coulnd't prefix delete: %v", err)
	}
	if count != 4 {
		t.Errorf("Expected DeletePrefix to remove 4 entries; only removed %d", count)
	}
	assert.NoError(t, badger.PrefixScan([]byte("key1"), func(key []byte, val []byte) error {
		iteration += "1"
		t.Errorf("Prefix delete didn't remove keys expected nothing got key '%s' ", key)
		t.Errorf("Prefix delete didn't remove values expected nothing got key '%s'", string(val))
		return nil

	}))
}
