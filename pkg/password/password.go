// Package password hashes passwords and verifies passwords against hashes.
package password

import (
	"crypto/fips140"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// pbkdf2Prefix begins a PBKDF2-HMAC-SHA256 hash in passlib's format:
	// $pbkdf2-sha256$<iterations>$<salt>$<key>
	pbkdf2Prefix = "$pbkdf2-sha256$"
	// bcryptPrefix begins every bcrypt hash, whatever its minor version.
	bcryptPrefix = "$2"

	// pbkdf2Iterations is the iteration count OWASP recommends for
	// PBKDF2-HMAC-SHA256.
	pbkdf2Iterations = 600_000
	// pbkdf2MinIterations is the minimum iteration count NIST SP 800-132
	// recommends. It is far below pbkdf2Iterations on purpose: a minimum exists
	// to reject hashes that are plainly weak, not to enforce best practice, and
	// OWASP's recommendation rises over time, so a minimum that followed it
	// would reject hashes that already work.
	pbkdf2MinIterations = 1_000
	// pbkdf2MaxIterations is the highest iteration count NIST SP 800-132
	// suggests, for especially critical keys. It bounds how long a password
	// check, which any unauthenticated login request triggers, can take.
	pbkdf2MaxIterations = 10_000_000
	// pbkdf2SaltLength is in bytes. It is also the minimum length FIPS 140-3
	// permits.
	pbkdf2SaltLength = 16
	// pbkdf2KeyLength is in bytes. passlib requires exactly this length.
	pbkdf2KeyLength = 32
)

// ab64 is passlib's "adapted base64" encoding: the standard alphabet with "."
// in place of "+", and no padding.
var ab64 = base64.NewEncoding(
	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789./",
).WithPadding(base64.NoPadding)

// Hash returns a PBKDF2-HMAC-SHA256 hash of password in passlib's
// "$pbkdf2-sha256$<iterations>$<salt>$<key>" format.
func Hash(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltLength)
	_, _ = rand.Read(salt) // Never returns an error
	key, err := pbkdf2.Key(
		sha256.New,
		password,
		salt,
		pbkdf2Iterations,
		pbkdf2KeyLength,
	)
	if err != nil {
		return "", fmt.Errorf("error deriving key: %w", err)
	}
	h := pbkdf2Hash{
		iterations: pbkdf2Iterations,
		salt:       salt,
		key:        key,
	}
	return h.String(), nil
}

// Verify reports whether password matches hash. hash may be a
// PBKDF2-HMAC-SHA256 hash in the format Hash returns, or a bcrypt hash. An
// error is returned if hash is in neither format, or if it is a bcrypt hash and
// Go's FIPS 140-3 mode is enabled.
func Verify(hash, password string) (bool, error) {
	h, err := parse(hash, fips140.Enabled())
	if err != nil {
		return false, err
	}
	return h.matches(password)
}

// Validate returns an error if Verify would return one for hash. Only checking
// a password against a hash fully establishes that the hash is usable, so
// Validate does that, and costs as much as a call to Verify.
func Validate(hash string) error {
	h, err := parse(hash, fips140.Enabled())
	if err != nil {
		return err
	}
	// A mismatch is expected and is not an error
	_, err = h.matches("")
	return err
}

// parsedHash is a password hash that has been parsed and is ready to check
// passwords against.
type parsedHash interface {
	// matches reports whether password matches the hash.
	matches(password string) (bool, error)
}

// parse parses hash, choosing its algorithm from its prefix. When fipsEnabled
// is true, hashes FIPS 140-3 does not permit are rejected: bcrypt hashes, since
// bcrypt is not a FIPS-approved algorithm, and PBKDF2 hashes with short salts.
func parse(hash string, fipsEnabled bool) (parsedHash, error) {
	switch {
	case strings.HasPrefix(hash, pbkdf2Prefix):
		return parsePBKDF2(hash, fipsEnabled)
	case strings.HasPrefix(hash, bcryptPrefix):
		if fipsEnabled {
			return nil, errors.New("bcrypt hashes are not permitted in FIPS 140-3 mode")
		}
		if _, err := bcrypt.Cost([]byte(hash)); err != nil {
			return nil, fmt.Errorf("error parsing bcrypt hash: %w", err)
		}
		return bcryptHash(hash), nil
	default:
		return nil, errors.New("hash is neither a PBKDF2-HMAC-SHA256 hash nor a bcrypt hash")
	}
}

// pbkdf2Hash is a parsed PBKDF2-HMAC-SHA256 hash.
type pbkdf2Hash struct {
	iterations int
	salt       []byte
	key        []byte
}

// parsePBKDF2 parses a PBKDF2-HMAC-SHA256 hash in passlib's format. When
// fipsEnabled is true, salts shorter than FIPS 140-3 permits are rejected.
func parsePBKDF2(hash string, fipsEnabled bool) (pbkdf2Hash, error) {
	fields := strings.Split(strings.TrimPrefix(hash, pbkdf2Prefix), "$")
	if len(fields) != 3 {
		return pbkdf2Hash{}, fmt.Errorf(
			"PBKDF2 hash does not have the form %s<iterations>$<salt>$<key>",
			pbkdf2Prefix,
		)
	}
	iterations, err := strconv.Atoi(fields[0])
	if err != nil {
		return pbkdf2Hash{}, fmt.Errorf(
			"PBKDF2 hash has invalid iteration count %q",
			fields[0],
		)
	}
	if iterations < pbkdf2MinIterations || iterations > pbkdf2MaxIterations {
		return pbkdf2Hash{}, fmt.Errorf(
			"PBKDF2 hash has iteration count %d; it must be between %d and %d",
			iterations, pbkdf2MinIterations, pbkdf2MaxIterations,
		)
	}
	salt, err := ab64.DecodeString(fields[1])
	if err != nil {
		return pbkdf2Hash{}, fmt.Errorf("error decoding PBKDF2 salt: %w", err)
	}
	if fipsEnabled && len(salt) < pbkdf2SaltLength {
		return pbkdf2Hash{}, fmt.Errorf(
			"PBKDF2 salts shorter than %d bytes are not permitted in FIPS 140-3 mode",
			pbkdf2SaltLength,
		)
	}
	key, err := ab64.DecodeString(fields[2])
	if err != nil {
		return pbkdf2Hash{}, fmt.Errorf("error decoding PBKDF2 key: %w", err)
	}
	if len(key) != pbkdf2KeyLength {
		return pbkdf2Hash{}, fmt.Errorf(
			"PBKDF2 key is %d bytes; expected %d",
			len(key), pbkdf2KeyLength,
		)
	}
	return pbkdf2Hash{
		iterations: iterations,
		salt:       salt,
		key:        key,
	}, nil
}

// matches implements parsedHash.
func (h pbkdf2Hash) matches(password string) (bool, error) {
	key, err := pbkdf2.Key(
		sha256.New,
		password,
		h.salt,
		h.iterations,
		len(h.key),
	)
	if err != nil {
		return false, fmt.Errorf("error deriving key: %w", err)
	}
	return subtle.ConstantTimeCompare(key, h.key) == 1, nil
}

// String returns h in passlib's format.
func (h pbkdf2Hash) String() string {
	return fmt.Sprintf(
		"%s%d$%s$%s",
		pbkdf2Prefix, h.iterations,
		ab64.EncodeToString(h.salt), ab64.EncodeToString(h.key),
	)
}

// bcryptHash is a bcrypt hash known to be well formed.
type bcryptHash []byte

// matches implements parsedHash.
func (h bcryptHash) matches(password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(h, []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error comparing bcrypt hash: %w", err)
	}
	return true, nil
}
