package idmapper_test

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/kvanticoss/gdpr-id-mapper/pkg/idmapper"
	badgerAdaptor "github.com/kvanticoss/gdpr-id-mapper/pkg/kvstore/badger"
	"github.com/kvanticoss/gdpr-id-mapper/pkg/record"

	"github.com/stretchr/testify/assert"
)

//nolint:unparam
func assertGoogdQueryRes(t *testing.T, gm *idmapper.GdprMapper, key [][]byte, ts *time.Duration) *record.PrivateRecord {
	rec, err := gm.Query(key, ts)
	assert.NoError(t, err, "Queries should always succeed")
	return rec
}

func TestIdmapperGDPRMapper(t *testing.T) {
	opts := badger.DefaultOptions("/tmp/badger2")
	opts.Truncate = true
	opts.SyncWrites = true
	opts.TableLoadingMode = options.LoadToRAM
	opts.ValueLogLoadingMode = options.FileIO
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatal(err.Error())
	}

	badger := badgerAdaptor.NewAdapter(db)
	defer badger.Close()

	gm := idmapper.NewGdprMapper(context.Background(), badger, []byte("almost random"), time.Minute)

	t.Run("Create and Read", func(t *testing.T) {

		// 1 level deep
		rec11 := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path1")}, nil)
		rec11b := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path1")}, nil)
		rec11.AliveUntil = rec11b.AliveUntil // Time are updated on query so they can't be expected to match
		assert.Equal(t, rec11, rec11b, "No ID should be like another")

		// 2 levels deep key
		rec12 := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path1"), []byte("subpath1")}, nil)
		rec12b := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path1"), []byte("subpath1")}, nil)
		rec12.AliveUntil = rec12b.AliveUntil // Time are updated on query so they can't be expected to match
		assert.Equal(t, rec11, rec11b, "No ID should be like another")

		// 2 level deep key; other base
		rec21 := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path2"), []byte("subpath1")}, nil)
		rec21b := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path2"), []byte("subpath1")}, nil)
		rec21.AliveUntil = rec21b.AliveUntil // Time are updated on query so they can't be expected to match
		assert.NotEqual(t, rec21, rec12, "No ID should be like another")
		assert.Equal(t, rec21, rec21b, "No ID should be like another")

		// 1 level deep key after doing a read on a sub-level
		rec11c := assertGoogdQueryRes(t, gm, [][]byte{[]byte("path1")}, nil)
		rec11.AliveUntil = rec11c.AliveUntil // Time are updated on query so they can't be expected to match
		assert.Equal(t, rec11c, rec11, "No ID should be like another")
	})

	t.Run("Create, Read, Wipe (tail), Read", func(t *testing.T) {
		// 1 level deep
		createdSub := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeTail"), []byte("sub")}, nil)
		createdBase := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeTail")}, nil)

		count, err := gm.ClearPrefix([][]byte{[]byte("wipeTail"), []byte("sub")})
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		recreatedSub := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeTail"), []byte("sub")}, nil)
		assert.NotEqual(t, createdSub, recreatedSub, "After a wipe the ids should not be the same")
		reloadedBase := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeTail")}, nil)
		createdBase.AliveUntil = reloadedBase.AliveUntil
		assert.Equal(t, createdBase, reloadedBase, "After a wipe on the sub the base should remain")
	})

	t.Run("Create, Read, Wipe(base), Read", func(t *testing.T) {
		// 1 level deep
		createdSub := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeBase"), []byte("sub")}, nil)
		createdBase := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeBase")}, nil)

		count, err := gm.ClearPrefix([][]byte{[]byte("wipeBase")})
		assert.NoError(t, err)
		assert.Equal(t, 2, count)

		recreatedSub := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeBase"), []byte("sub")}, nil)
		assert.NotEqual(t, createdSub, recreatedSub, "After a wipe the ids should not be the same")
		reloadedBase := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeBase")}, nil)
		createdBase.AliveUntil = reloadedBase.AliveUntil
		assert.NotEqual(t, createdBase, reloadedBase, "After a wipe on the base the base should be different")
	})

	t.Run("Set; Read", func(t *testing.T) {
		// 1 level deep
		tailID, err := gm.Set([][]byte{[]byte("wipeBase"), []byte("sub")}, []byte("ExistingId"), nil)
		assert.NoError(t, err)
		loadedTail := assertGoogdQueryRes(t, gm, [][]byte{[]byte("wipeBase"), []byte("sub")}, nil)
		tailID.AliveUntil = loadedTail.AliveUntil
		assert.Equal(t, tailID, loadedTail, "After an import(set) the ids should be the same")
	})
}
