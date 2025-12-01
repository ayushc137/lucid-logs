package database

import (
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// SurrealTime handles datetime fields returned by SurrealDB.
// SurrealDB can encode datetimes as:
//   - Native CBOR datetime tags (decoded into time.Time)
//   - Raw arrays [seconds, nanoseconds] for custom datetime tags
//
// This type gracefully handles those formats and normalizes to UTC.
type SurrealTime struct {
	time.Time
}

// TimeValue returns the underlying time.
func (st SurrealTime) TimeValue() time.Time {
	return st.Time
}

// Ptr returns a pointer to the underlying time or nil if zero.
func (st SurrealTime) Ptr() *time.Time {
	if st.Time.IsZero() {
		return nil
	}
	t := st.Time
	return &t
}

// UnmarshalCBOR implements cbor.Unmarshaler so we can handle the various
// datetime encodings SurrealDB may send back.
func (st *SurrealTime) UnmarshalCBOR(data []byte) error {
	// Fast path: SurrealDB tagged datetime that decodes directly into time.Time
	var asTime time.Time
	if err := cbor.Unmarshal(data, &asTime); err == nil {
		st.Time = asTime.UTC()
		return nil
	}

	// Custom datetime tag may present as raw [seconds, nanoseconds]
	var arr []int64
	if err := cbor.Unmarshal(data, &arr); err == nil && len(arr) == 2 {
		st.Time = time.Unix(arr[0], arr[1]).UTC()
		return nil
	}

	return fmt.Errorf("unsupported SurrealDB time value")
}
