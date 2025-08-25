// apps/go-server/internal/daily/daily.go
//
// Helpers for the "Daily Challenge" feature.
// Provides deterministic mapping from dates to word indices,
// ensuring that all players see the same solution word on a given date
// (while allowing server operators to rotate the mapping with a secret salt).

package daily

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

/**
 * DateKey returns a normalized string representation of a date.
 *
 * - Always UTC.
 * - Format: "YYYY-MM-DD"
 *
 * Example: 2025-08-24 15:32 UTC → "2025-08-24"
 */
func DateKey(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

/**
 * WordIndex produces a deterministic index into the answers list for a given date.
 *
 * Implementation:
 *   - Normalize the date to "YYYY-MM-DD" via DateKey.
 *   - Compute HMAC-SHA256 of that string using the provided salt.
 *   - Take the first 8 bytes of the digest as a uint64.
 *   - Return (digest % answersLen) as the index.
 *
 * This ensures:
 *   - Same date → same index for all players.
 *   - Changing the salt → rotates the mapping (useful if word list order is known).
 *
 * @param date        Date for which to compute index.
 * @param salt        Secret string that personalizes HMAC; should be constant server-side.
 * @param answersLen  Length of answers list (must be > 0).
 * @return int index in [0, answersLen).
 */
func WordIndex(date time.Time, salt string, answersLen int) int {
	if answersLen <= 0 {
		return 0
	}
	dk := DateKey(date)
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(dk))
	sum := h.Sum(nil)

	// Use first 8 bytes → uint64 for uniform modulus distribution.
	n := binary.BigEndian.Uint64(sum[:8])
	return int(n % uint64(answersLen))
}
