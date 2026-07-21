# Database Package

This package provides the libSQL (Turso) connection and query utilities with type-safe operations on top of `database/sql`.

## RecordID Convention

**Use `models.RecordID` for any struct field that stores a record ID in database-facing structs. Convert to string only when crossing API boundaries.**

Record IDs use the `table:value` format (e.g. `tasks:abc123`). They are stored as plain TEXT primary keys in SQLite, so the driver returns them as strings — but wrapping them in `models.RecordID` keeps the domain layer explicit about the format and validates it on unmarshal.

### Pattern

```go
// ✅ GOOD: Uses models.RecordID for DB struct
type taskDB struct {
    ID models.RecordID `json:"id,omitempty"`
    // ...
}

// Query is simple - the row mapper handles deserialization
results, _ := database.QueryAll[taskDB](ctx, db, `
    SELECT * FROM tasks
`, nil)

// Convert at boundary
func (t *taskDB) toTask() *Task {
    return &Task{
        ID: database.ToStringID(t.ID),  // Convert here
        // ...
    }
}
```

## Record Links (Relationships)

For fields that reference other records (like `task.category_id`), use `models.RecordID`:

```go
type taskCreateData struct {
    CategoryID *models.RecordID `json:"category_id,omitempty"` // Record link
    // ...
}

// Create the link
categoryLink := database.NewRecordID("categories", categoryID)
data := taskCreateData{
    CategoryID: &categoryLink,
}
```

## Helper Functions

| Function | Purpose |
|----------|---------|
| `ToStringID(rid models.RecordID)` | Convert RecordID to string for API responses |
| `MustRecordID(table, raw string)` | Create RecordID from table + ID string |
| `NewRecordID(table string, id any)` | Create RecordID for record links |
| `RecordID(table, id string)` | Format string record ID (legacy helper) |
| `RecordIDFromString(s string)` | Parse string to RecordID with error handling |

## Struct Naming Convention

- `*DB` suffix: Database-facing structs with `models.RecordID` (e.g., `taskDB`, `categoryDB`, `userDB`)
- No suffix: Domain/API-facing structs with string IDs (e.g., `Task`, `Category`, `User`)

## Timestamps

Timestamp columns are stored as RFC3339 TEXT. Use `database.FlexTime` in scan structs — it embeds `time.Time` and provides `Ptr()` for nullable columns.

## Example Flow

```
API Request
    ↓
Handler (string IDs)
    ↓
Service (string IDs)
    ↓
Repository.Create()
    → MustRecordID() to create RecordID
    → Query with RecordID
    → Result as *DB struct with RecordID
    → toModel() converts to domain struct with string ID
    ↓
API Response (string IDs)
```
