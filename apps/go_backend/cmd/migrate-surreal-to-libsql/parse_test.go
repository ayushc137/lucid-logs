package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseExport_BundleWithTablesKey(t *testing.T) {
	path := writeFixture(t, "bundle.json", `{
		"tables": {
			"users": [{"id": "users:u1", "email": "a@b.c", "pass": "hash"}],
			"tasks": [
				{"id": "tasks:t1", "title": "One"},
				{"id": "tasks:t2", "title": "Two"}
			]
		}
	}`)
	exp, err := ParseExport(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(exp.Tables["users"]); got != 1 {
		t.Errorf("users rows = %d, want 1", got)
	}
	if got := len(exp.Tables["tasks"]); got != 2 {
		t.Errorf("tasks rows = %d, want 2", got)
	}
}

func TestParseExport_TopLevelTableKeys(t *testing.T) {
	path := writeFixture(t, "tables.json", `{
		"goals": [{"id": "goals:g1", "title": "Goal"}],
		"units": []
	}`)
	exp, err := ParseExport(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(exp.Tables["goals"]); got != 1 {
		t.Errorf("goals rows = %d, want 1", got)
	}
	if _, ok := exp.Tables["units"]; !ok {
		t.Errorf("expected empty units table to be present")
	}
}

func TestParseExport_SurrealCLIEnvelope(t *testing.T) {
	path := writeFixture(t, "cli.json", `[
		{"status": "OK", "time": "1.2ms", "result": [
			{"id": "emotions:e1", "name": "Joy"},
			{"id": "emotions:e2", "name": "Calm"}
		]},
		{"status": "OK", "time": "0.5ms", "result": [
			{"id": "categories:c1", "name": "Work"}
		]}
	]`)
	exp, err := ParseExport(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(exp.Tables["emotions"]); got != 2 {
		t.Errorf("emotions rows = %d, want 2", got)
	}
	if got := len(exp.Tables["categories"]); got != 1 {
		t.Errorf("categories rows = %d, want 1", got)
	}
}

func TestParseExport_BareRowArray(t *testing.T) {
	path := writeFixture(t, "rows.json", `[
		{"id": "units:u1", "name": "km"},
		{"id": "units:u2", "name": "kg"}
	]`)
	exp, err := ParseExport(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(exp.Tables["units"]); got != 2 {
		t.Errorf("units rows = %d, want 2", got)
	}
}

func TestParseExport_NDJSON(t *testing.T) {
	path := writeFixture(t, "rows.ndjson", "{\"id\": \"goals:g1\"}\n{\"id\": \"goals:g2\"}\n\n")
	exp, err := ParseExport(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(exp.Tables["goals"]); got != 2 {
		t.Errorf("goals rows = %d, want 2", got)
	}
}

func TestParseExport_Errors(t *testing.T) {
	if _, err := ParseExport(writeFixture(t, "empty.json", "  ")); err == nil {
		t.Error("expected error for empty file")
	}
	// Row without record ID can't be table-inferred.
	if _, err := ParseExport(writeFixture(t, "bad.json", `[{"title": "no id"}]`)); err == nil {
		t.Error("expected error for rows without ids")
	}
	// Failed statement in CLI envelope.
	if _, err := ParseExport(writeFixture(t, "fail.json", `[{"status":"ERR","result":[]}]`)); err == nil {
		t.Error("expected error for failed statement")
	}
}

func TestHelpers(t *testing.T) {
	row := Row{
		"name":     "test",
		"priority": float64(1),
		"enabled":  true,
		"nested":   map[string]any{"a": float64(1)},
		"empty":    "",
		"nilval":   nil,
	}
	if got := str(row, "name"); got != "test" {
		t.Errorf("str = %q", got)
	}
	if got := str(row, "nilval"); got != "" {
		t.Errorf("str(nil) = %q, want empty", got)
	}
	if got := boolInt(row, "enabled"); got != 1 {
		t.Errorf("boolInt = %d, want 1", got)
	}
	if got := priorityLabel(row, "priority"); got == nil || *got != "high" {
		t.Errorf("priorityLabel(1) = %v, want high", got)
	}
	row["priority"] = float64(3)
	if got := priorityLabel(row, "priority"); got == nil || *got != "low" {
		t.Errorf("priorityLabel(3) = %v, want low", got)
	}
	row["priority"] = "medium"
	if got := priorityLabel(row, "priority"); got == nil || *got != "medium" {
		t.Errorf("priorityLabel(medium) = %v, want medium", got)
	}
	if got := jsonText(row, "nested"); got == nil || *got != `{"a":1}` {
		t.Errorf("jsonText = %v", got)
	}
	if got := jsonText(row, "missing"); got != nil {
		t.Errorf("jsonText(missing) = %v, want nil", got)
	}
	if got := jsonTextOr(row, "missing", "{}"); got != "{}" {
		t.Errorf("jsonTextOr = %q, want {}", got)
	}
	if got := intVal(row, "priority"); got != 0 {
		t.Errorf("intVal(string) = %d, want 0", got)
	}
	row["num"] = float64(7)
	if got := intVal(row, "num"); got != 7 {
		t.Errorf("intVal = %d, want 7", got)
	}
}
