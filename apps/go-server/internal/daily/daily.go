package daily

import (
"crypto/hmac"
"crypto/sha256"
"encoding/binary"
"time"
)

// DateKey returns YYYY-MM-DD in UTC.
func DateKey(t time.Time) string {
return t.UTC().Format("2006-01-02")
}

// WordIndex returns a deterministic index for a date using HMAC(salt, YYYY-MM-DD) % answersLen.
func WordIndex(date time.Time, salt string, answersLen int) int {
if answersLen <= 0 {
return 0
}
dk := DateKey(date)
h := hmac.New(sha256.New, []byte(salt))
h.Write([]byte(dk))
sum := h.Sum(nil)
// take first 8 bytes to uint64 for modulus distribution
n := binary.BigEndian.Uint64(sum[:8])
return int(n % uint64(answersLen))
}