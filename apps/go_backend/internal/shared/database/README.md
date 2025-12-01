# Database Package

This package provides SurrealDB connection and query utilities with type-safe operations.

## RecordID Convention

**Use `models.RecordID` for any struct field that stores SurrealDB record IDs in database-facing structs. Convert to string only when crossing API boundaries.**

### Why?

SurrealDB returns record IDs as structured objects (`{ tb: "tasks", id: "abc123" }`), not strings. Using `models.RecordID` in DB structs allows the SDK to deserialize IDs correctly without query-time casts like `type::string(id) AS id`.

### Pattern

```go
// ❌ BAD: Uses string for DB struct, requires type::string cast
type taskDB struct {
    ID string `json:"id"`
    // ...
}

// Query needs explicit cast
results, _ := database.QueryAll[taskDB](ctx, db, `
    SELECT *, type::string(id) AS id FROM tasks
`, nil)

// ✅ GOOD: Uses models.RecordID for DB struct
type taskDB struct {
    ID models.RecordID `json:"id,omitempty"`
    // ...
}

// Query is simple - SDK handles deserialization
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

For fields that reference other records (like `task.category`), use `models.RecordID`:

```go
type taskCreateData struct {
    Category *models.RecordID `json:"category,omitempty"` // Record link
    // ...
}

// Create the link
categoryLink := database.NewRecordID("categories", categoryID)
data := taskCreateData{
    Category: &categoryLink,
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

## Benefits

1. **No query-time casts**: Plain `SELECT *` instead of `SELECT *, type::string(id) AS id`
2. **Type safety**: SDK enforces RecordID shape at compile time
3. **Simpler queries**: No custom RETURN blocks to remap IDs
4. **Consistent links**: `models.NewRecordID()` ensures valid record references
5. **Future-proof**: Won't forget casts when adding new queries

