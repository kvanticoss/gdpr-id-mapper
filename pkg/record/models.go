package record

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// SaltLength is the amount random bytes to read into the salt
const SaltLength = 32

// PrivateRecord contains the contents that all cost must be keep secure to
// guarrantee the integrity of the system
type PrivateRecord struct {
	OriginalID []byte
	Salt       []byte
	PublicID   []byte
	AliveUntil time.Time
}

// PublicResultRecord is a subset of PrivateRecord prepared for export; as such the hash is hex encoded and salt removed
type PublicResultRecord struct {
	OriginalID string
	PublicID   string
	AliveUntil time.Time
}

// NewPrivateRecord create an private record instace with a random Salt and sha256 hash of the salt + id
func NewPrivateRecord(id []byte, aliveUntil time.Time) *PrivateRecord {
	salt := randomBytes()
	hash := sha256.Sum256(append(salt, id...))
	return &PrivateRecord{
		OriginalID: id,
		Salt:       salt,
		PublicID:   hash[:],
		AliveUntil: aliveUntil,
	}
}

// PublicVersion returns a record containing the internal id and the public representation of it
func (record *PrivateRecord) PublicVersion() *PublicResultRecord {
	return &PublicResultRecord{
		OriginalID: base64.URLEncoding.EncodeToString(record.OriginalID),
		PublicID:   base64.URLEncoding.EncodeToString(record.PublicID),
		AliveUntil: record.AliveUntil,
	}
}

// Encode marshals the record contents into a byte array
func (record *PrivateRecord) Encode() ([]byte, error) {
	return json.Marshal(record)
}

// Decode unmarshals the the byte-array into the current record
func (record *PrivateRecord) Decode(buffer []byte) error {
	return json.Unmarshal(buffer, record)
}

func randomBytes() []byte {
	randBytes := make([]byte, SaltLength)
	if _, err := rand.Read(randBytes); err != nil {
		panic(err)
	}
	return randBytes
}
