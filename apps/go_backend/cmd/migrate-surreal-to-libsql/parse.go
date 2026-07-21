package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Row is a single SurrealDB record decoded from the JSON export.
type Row map[string]any

// Export is the normalized in-memory representation of a SurrealDB export:
// table name -> rows.
type Export struct {
	Tables map[string][]Row
}

// surrealEnvelope matches the top-level shape of `surreal sql --json` output:
// an array of statement results, each with status/result fields.
type surrealEnvelope struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Time   string          `json:"time"`
}

// ParseExport reads a SurrealDB JSON export and normalizes it into an Export.
//
// Supported input shapes:
//
//  1. Bundle:    {"tables": {"tasks": [ {...}, ... ], "goals": [...]}}
//  2. Bundle:    {"tasks": [...], "goals": [...]}            (table-name keys at top level)
//  3. Raw CLI:   [{"status":"OK","result":[{...}, ...]}]     (single `surreal sql --json`
//     query result for one table; table inferred from record IDs)
//  4. Raw array: [ {...}, ... ]                              (bare row array, table inferred)
//
// Rows may also appear as newline-delimited JSON objects (one per line) when
// the file is not valid JSON as a whole — table inferred per row.
func ParseExport(path string) (*Export, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read export: %w", err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("export file is empty")
	}

	exp := &Export{Tables: map[string][]Row{}}

	switch trimmed[0] {
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			// A single-JSON-object parse failure with multiple lines likely means
			// newline-delimited JSON — fall through to the NDJSON path.
			if strings.Contains(trimmed, "\n") {
				return parseNDJSON(trimmed)
			}
			return nil, fmt.Errorf("parse JSON object: %w", err)
		}
		// Shape 1: {"tables": {...}}
		if raw, ok := obj["tables"]; ok {
			var tables map[string][]Row
			if err := json.Unmarshal(raw, &tables); err != nil {
				return nil, fmt.Errorf("parse \"tables\" key: %w", err)
			}
			for name, rows := range tables {
				exp.Tables[name] = append(exp.Tables[name], rows...)
			}
			return exp, nil
		}
		// Shape 2: table-name keys at top level.
		for key, raw := range obj {
			var rows []Row
			if err := json.Unmarshal(raw, &rows); err != nil {
				return nil, fmt.Errorf("top-level key %q is not a row array (expected a tables bundle)", key)
			}
			exp.Tables[key] = append(exp.Tables[key], rows...)
		}
		return exp, nil

	case '[':
		// Try raw CLI envelope first.
		var env []surrealEnvelope
		if err := json.Unmarshal(data, &env); err == nil && len(env) > 0 && env[0].Status != "" {
			for _, stmt := range env {
				if stmt.Status != "OK" {
					return nil, fmt.Errorf("surreal export contains failed statement: %s", stmt.Status)
				}
				var rows []Row
				if err := json.Unmarshal(stmt.Result, &rows); err != nil {
					// Some statements return a single object or null — skip non-arrays.
					continue
				}
				if err := exp.addInferred(rows); err != nil {
					return nil, err
				}
			}
			return exp, nil
		}
		// Bare row array.
		var rows []Row
		if err := json.Unmarshal(data, &rows); err != nil {
			return nil, fmt.Errorf("parse JSON array: %w", err)
		}
		if err := exp.addInferred(rows); err != nil {
			return nil, err
		}
		return exp, nil

	default:
		return parseNDJSON(trimmed)
	}
}

// parseNDJSON handles newline-delimited JSON objects (one per line).
func parseNDJSON(trimmed string) (*Export, error) {
	exp := &Export{Tables: map[string][]Row{}}
	lines := strings.Split(trimmed, "\n")
	var rows []Row
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row Row
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("line %d: not valid JSON: %w", i+1, err)
		}
		rows = append(rows, row)
	}
	if err := exp.addInferred(rows); err != nil {
		return nil, err
	}
	return exp, nil
}

// addInferred appends rows to Tables keyed by the table part of each row's
// Surreal record ID ("tasks:abc123" -> "tasks").
func (e *Export) addInferred(rows []Row) error {
	for _, row := range rows {
		id, ok := row["id"].(string)
		if !ok || !strings.Contains(id, ":") {
			return fmt.Errorf("cannot infer table for row without a record id (id=%v); use a bundle with explicit table names", row["id"])
		}
		table, _, _ := strings.Cut(id, ":")
		e.Tables[table] = append(e.Tables[table], row)
	}
	return nil
}

// --- Field extraction helpers ----------------------------------------------

func str(row Row, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// strPtr returns nil for missing/null/empty values.
func strPtr(row Row, key string) *string {
	s := str(row, key)
	if s == "" {
		return nil
	}
	return &s
}

func floatPtr(row Row, key string) *float64 {
	v, ok := row[key]
	if !ok || v == nil {
		return nil
	}
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	return &f
}

func floatVal(row Row, key string) float64 {
	if f := floatPtr(row, key); f != nil {
		return *f
	}
	return 0
}

func intVal(row Row, key string) int64 {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	}
	return 0
}

func boolInt(row Row, key string) int64 {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	b, ok := v.(bool)
	if !ok {
		return 0
	}
	if b {
		return 1
	}
	return 0
}

// jsonText marshals a nested Surreal value (object or array) to JSON text.
// Returns nil for missing/null values so the column stays NULL.
func jsonText(row Row, key string) *string {
	v, ok := row[key]
	if !ok || v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// jsonTextOr marshals like jsonText but returns the given default when the
// key is missing/null (for NOT NULL columns with defaults).
func jsonTextOr(row Row, key, def string) string {
	if s := jsonText(row, key); s != nil {
		return *s
	}
	return def
}

// priorityLabel maps the legacy 1-3 integer priority to the new label.
func priorityLabel(row Row, key string) *string {
	v, ok := row[key]
	if !ok || v == nil {
		return nil
	}
	// Already a label?
	if s, ok := v.(string); ok {
		switch strings.ToLower(s) {
		case "high", "medium", "low":
			l := strings.ToLower(s)
			return &l
		}
		return nil
	}
	n, ok := v.(float64)
	if !ok {
		return nil
	}
	var label string
	switch {
	case n <= 1:
		label = "high"
	case n == 2:
		label = "medium"
	default:
		label = "low"
	}
	return &label
}
