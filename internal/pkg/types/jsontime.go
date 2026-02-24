package types

import (
	"encoding/json"
	"strings"
	"time"
)

// JSONTime is a custom time type that handles multiple JSON date formats
type JSONTime struct {
	time.Time
}

func (jt *JSONTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), "\"")
	if s == "null" || s == "" {
		return nil
	}

	// Try RFC3339 first (standard for time.Time)
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		jt.Time = t
		return nil
	}

	// Try ISO8601/Date only format
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		jt.Time = t
		return nil
	}

	// Try date with time but without Z
	t, err = time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		jt.Time = t
		return nil
	}

	_, err = time.Parse(time.RFC3339, s)
	return err
}

func (jt JSONTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(jt.Format(time.RFC3339))
}
