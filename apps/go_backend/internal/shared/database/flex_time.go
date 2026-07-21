package database

import (
	"time"
)

// FlexTime is a time.Time wrapper used by database-facing structs.
//
// Timestamps are stored in the local libSQL database as RFC3339 strings, so a
// plain time.Time would suffice for JSON round-tripping. The wrapper exists to
// keep repository scan structs explicit about which columns carry timestamps
// and to provide the convenience helpers below.
type FlexTime struct {
	time.Time
}

// TimeValue returns the underlying time.
func (st FlexTime) TimeValue() time.Time {
	return st.Time
}

// Ptr returns a pointer to the underlying time or nil if zero.
func (st FlexTime) Ptr() *time.Time {
	if st.Time.IsZero() {
		return nil
	}
	t := st.Time
	return &t
}
