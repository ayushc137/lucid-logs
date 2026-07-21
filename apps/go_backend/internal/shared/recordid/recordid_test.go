package models

import (
	"encoding/json"
	"testing"
)

func TestRecordIDRoundTrip(t *testing.T) {
	id := NewRecordID("tasks", "abc")
	if got := id.String(); got != "tasks:abc" {
		t.Fatalf("String() = %q, want tasks:abc", got)
	}

	encoded, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `"tasks:abc"` {
		t.Fatalf("MarshalJSON() = %s", encoded)
	}

	var decoded RecordID
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.String() != id.String() {
		t.Fatalf("decoded = %q, want %q", decoded.String(), id.String())
	}
}

func TestParseRecordIDRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"", "tasks", ":abc", "tasks:"} {
		if _, err := ParseRecordID(value); err == nil {
			t.Fatalf("ParseRecordID(%q) unexpectedly succeeded", value)
		}
	}
}
