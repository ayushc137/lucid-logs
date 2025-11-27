# {{feature_name_pascal}} Feature Template

This template was generated for the `{{feature_name}}` feature.

## Generated Files

```
src/
├── handlers/{{feature_name}}.rs     # HTTP endpoints
├── models/{{feature_name}}.rs       # Entity and DTOs
├── services/{{feature_name}}.rs     # Business logic
└── repositories/{{feature_name}}.rs # Database operations
```

## Integration Steps

After generating, you need to integrate the feature:

### 1. Add to Module Exports

**`src/handlers/mod.rs`**:
```rust
pub mod {{feature_name}};
```

**`src/models/mod.rs`**:
```rust
pub mod {{feature_name}};
```

**`src/services/mod.rs`**:
```rust
pub mod {{feature_name}};
pub use {{feature_name}}::{{feature_name_pascal}}ServiceImpl;
```

**`src/repositories/mod.rs`**:
```rust
pub mod {{feature_name}};
pub use {{feature_name}}::{{feature_name_pascal}}Repository;
```

### 2. Add Service Trait

In `src/services/traits.rs`, add the trait (or copy from the generated service file):
```rust
#[async_trait]
pub trait {{feature_name_pascal}}Service: Send + Sync {
    // ... methods
}
```

### 3. Update AppState

In `src/utils/state.rs`:
```rust
pub struct AppState {
    // ... existing fields
    pub {{feature_name}}_service: Arc<dyn {{feature_name_pascal}}Service>,
}
```

### 4. Wire Up in main.rs

```rust
// Create service
let {{feature_name}}_service = Arc::new({{feature_name_pascal}}ServiceImpl::new(db.clone()));

// Add to AppState
let app_state = AppState::new(
    db,
    settings.clone(),
    task_service,
    auth_service,
    {{feature_name}}_service,  // Add here
);

// Add routes
let api_v1 = Router::new()
    .merge(handlers::health::routes())
    .merge(handlers::auth::routes())
    .merge(handlers::task::protected_routes(app_state.clone()))
    .merge(handlers::{{feature_name}}::protected_routes(app_state.clone()));  // Add here
```

### 5. Add Database Schema

Create migration in `db/migrations/` or add to `db/schema.surql`:
```surql
DEFINE TABLE {{table_name}} SCHEMAFULL;
DEFINE FIELD name ON {{table_name}} TYPE string;
DEFINE FIELD description ON {{table_name}} TYPE option<string>;
DEFINE FIELD created_at ON {{table_name}} TYPE datetime;
DEFINE FIELD updated_at ON {{table_name}} TYPE datetime;
DEFINE FIELD deleted_at ON {{table_name}} TYPE option<datetime>;
DEFINE FIELD created_by ON {{table_name}} TYPE string;
DEFINE FIELD updated_by ON {{table_name}} TYPE string;

DEFINE INDEX {{table_name}}_owner ON {{table_name}} FIELDS created_by;
```

### 6. Update OpenAPI

In `main.rs`, add to the `ApiDoc` struct:
```rust
#[derive(OpenApi)]
#[openapi(
    paths(
        // ... existing paths
        handlers::{{feature_name}}::list_{{feature_name}}s,
        handlers::{{feature_name}}::get_{{feature_name}},
        handlers::{{feature_name}}::create_{{feature_name}},
        handlers::{{feature_name}}::update_{{feature_name}},
        handlers::{{feature_name}}::delete_{{feature_name}},
    ),
    components(
        schemas(
            // ... existing schemas
            models::{{feature_name}}::{{feature_name_pascal}},
            models::{{feature_name}}::Create{{feature_name_pascal}}Request,
            models::{{feature_name}}::Update{{feature_name_pascal}}Request,
        )
    ),
    // ... rest
)]
```

## Testing

Add tests in `tests/integration/{{feature_name}}_tests.rs`:
```rust
// See tests/integration/health_tests.rs for examples
```

## Customization

The generated code includes `todo!()` macros where you need to fill in implementation details. Remove these as you implement each method.

