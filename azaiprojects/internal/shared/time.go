package shared

import (
	"fmt"
	"strconv"
	"time"
)

// UnixSeconds is a time.Time that round-trips JSON as Unix seconds (the wire
// format used by the beta APIs for created_at fields).
type UnixSeconds struct {
	time.Time
}

// MarshalJSON renders the time as integer Unix seconds.
func (t UnixSeconds) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(t.Time.Unix(), 10)), nil
}

// UnmarshalJSON accepts an integer or float number of Unix seconds, or null.
func (t *UnixSeconds) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("UnixSeconds: %w", err)
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	t.Time = time.Unix(sec, nsec).UTC()
	return nil
}
