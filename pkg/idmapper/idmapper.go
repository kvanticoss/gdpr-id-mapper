package idmapper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/kvanticoss/gdpr-id-mapper/pkg/kvstore"
	"github.com/kvanticoss/gdpr-id-mapper/pkg/record"
	"github.com/kvanticoss/goutils/fdbtuple"
)

var (
	// ErrRecordNotFound is returned when trying to read or delete a key that doesn't exist
	ErrRecordNotFound = fmt.Errorf("record not found")

	// ErrNoPrefixProvided is returned when a whipe operation is conducted with no prefix pattern.
	ErrNoPrefixProvided = fmt.Errorf("no prefix provided")
)

// GdprMapper holds all state needed for id-mapping
type GdprMapper struct {
	ctx            context.Context
	db             kvstore.Store
	globalSalt     []byte
	deafultLiveTTL time.Duration
}

// NewGdprMapper instanciates a new GDPR mapper
func NewGdprMapper(
	ctx context.Context,
	db kvstore.Store,
	globalSalt []byte,
	defaultTTL time.Duration,
) *GdprMapper {
	return &GdprMapper{
		ctx:            ctx,
		db:             db,
		globalSalt:     globalSalt,
		deafultLiveTTL: defaultTTL,
	}
}

// Query searches for a leaf id and creates all ids that are missing
func (gm *GdprMapper) Query(ids [][]byte, ttl *time.Duration) (*record.PrivateRecord, error) {

	var rec *record.PrivateRecord
	var err error

	hashes := gm.getHierarchicalHash(ids)
	for index := range hashes {
		//log.Printf("Hash at index %d for %v is hashes:%x Packed: %x", index, ids, hashes[:index+1], string(hashes[:index+1].Pack()))
		rec, err = gm.getOrCreateSingleRecord(hashes[:index+1].Pack(), ttl)
		if err != nil {
			return nil, err
		}
	}

	//The last record created is the results we are interested in
	return rec, nil
}

// Set manually overrides the salt+id generated version and imports an historic id
func (gm *GdprMapper) Set(id [][]byte, publicID []byte, liveTTL *time.Duration) (*record.PrivateRecord, error) {
	r, err := gm.Query(id, liveTTL)
	if err != nil {
		return nil, err
	}
	r.PublicID = publicID
	recordBytes, err := r.Encode()
	if err != nil {
		return nil, err
	}
	return r, gm.db.Put(gm.getHierarchicalHash(id).Pack(), recordBytes, liveTTL)
}

// ClearPrefix removes all keys from the cache that starts with the elements in the key
func (gm *GdprMapper) ClearPrefix(prefixes [][]byte) (int, error) {
	if len(prefixes) == 0 {
		return 0, ErrNoPrefixProvided
	}

	keyPrefix := gm.getHierarchicalHash(prefixes).Pack()
	return gm.db.DeletePrefix(keyPrefix)
}

func (gm *GdprMapper) getOrCreateSingleRecord(hashedKey []byte, ttl *time.Duration) (*record.PrivateRecord, error) {
	existingRecord, err := gm.findExistingRecord(hashedKey, ttl)
	if err == nil {
		return existingRecord, err
	}
	if err == ErrRecordNotFound {
		return gm.createNewRecord(hashedKey, ttl)
	}
	return nil, err
}

func (gm *GdprMapper) createNewRecord(hashedKey []byte, ttl *time.Duration) (*record.PrivateRecord, error) {
	if ttl == nil {
		ttl = &gm.deafultLiveTTL
	}

	t := time.Now().Add(*ttl)
	newRec := record.NewPrivateRecord(hashedKey, t)
	recordBytes, err := newRec.Encode()
	if err != nil {
		return nil, err
	}

	return newRec, gm.db.Put(hashedKey, recordBytes, ttl)
}

func (gm *GdprMapper) findExistingRecord(hashedKey []byte, ttl *time.Duration) (*record.PrivateRecord, error) {
	//Try to find a live record
	tmpRecord := record.NewPrivateRecord([]byte("tmp"), time.Now())
	liveRecordBytes := gm.db.Get(hashedKey)
	if len(liveRecordBytes) == 0 {
		return nil, ErrRecordNotFound
	}
	if err := tmpRecord.Decode(liveRecordBytes); err != nil {
		return nil, err
	}
	// Have the record expired (but not yet been GC:ed?); Technically they shouldn't exist and we will ignore them
	// This is just a backup in case the DB layer fails.
	if tmpRecord.AliveUntil.Before(time.Now()) {
		_, _ = gm.db.Delete(hashedKey)
		return nil, ErrRecordNotFound
	}

	if ttl == nil {
		return tmpRecord, nil
	}

	tmpRecord.AliveUntil = time.Now().Add(*ttl)
	recordBytes, err := tmpRecord.Encode()
	if err != nil {
		return nil, err
	}
	return tmpRecord, gm.db.Put(hashedKey, recordBytes, ttl) // this is an simple update of the TTL; ok to ignore the error; tough not great
}

// getHierarchicalHash returns the hash of each byte slice salted with the hashes of the all the previous elements.
func (gm *GdprMapper) getHierarchicalHash(input [][]byte) fdbtuple.Tuple {
	hashes := fdbtuple.Tuple{}

	// Initial condition/salt; nada
	previousHash := make([]byte, 0, 32)
	for _, i := range input {
		previousHash = gm.hashByteWithInstanceSalt(append(previousHash, i...))
		hashes = append(hashes, previousHash)
	}
	return hashes
}

func (gm *GdprMapper) hashByteWithInstanceSalt(input []byte) []byte {
	hashArr := sha256.Sum256(append(gm.globalSalt, input...))
	return hashArr[:]
}
