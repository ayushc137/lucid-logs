// Package models provides the table:value record identifier used by Lucid Logs.
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// RecordID preserves the public table:value identifier contract without tying
// domain models to a database driver.
type RecordID struct {
	table string
	id    string
}

func NewRecordID(table string, id any) RecordID {
	return RecordID{table: table, id: fmt.Sprint(id)}
}

func ParseRecordID(value string) (*RecordID, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid record ID %q", value)
	}
	id := NewRecordID(parts[0], parts[1])
	return &id, nil
}

func (r RecordID) String() string {
	if r.table == "" {
		return r.id
	}
	return r.table + ":" + r.id
}

func (r RecordID) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// Value implements driver.Valuer so RecordID can be passed directly to
// database/sql as a query parameter (stored as "table:id" string).
func (r RecordID) Value() (driver.Value, error) {
	return r.String(), nil
}

func (r *RecordID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseRecordID(value)
	if err != nil {
		return err
	}
	*r = *parsed
	return nil
}
