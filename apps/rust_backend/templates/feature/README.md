# {{feature_name_pascal}} Feature Template

This template was generated for the `{{feature_name}}` feature using the new **feature-based (vertical slice)** architecture.

## Generated Files

```
src/features/{{feature_name}}/
├── mod.rs           # Module exports
├── handler.rs       # HTTP endpoints
├── model.rs         # Entity and DTOs
├── service.rs       # Business logic + trait
└── repository.rs    # Database operations
```

## Integration Steps

After generating, you need to integrate the feature:

### 1. Register the Feature Module

**`src/features/mod.rs`**:
```rust
pub mod {{feature_name}};

// Add to re-exports
pub use {{feature_name}}::{protected_routes as {{feature_name}}_protected_routes, routes as {{feature_name}}_routes};
pub use {{feature_name}}::{{{feature_name_pascal}}Service, {{feature_name_pascal}}ServiceImpl};
```

### 2. Update AppState

In `src/state.rs`:
```rust
use crate::features::{{feature_name_pascal}}Service;

pub struct AppState {
    // ... existing fields
    pub {{feature_name}}_service: Arc<dyn {{feature_name_pascal}}Service>,
}

impl AppState {
    pub fn new(
        // ... existing params
        {{feature_name}}_service: Arc<dyn {{feature_name_pascal}}Service>,
    ) -> Self {
        Self {
            // ... existing fields
            {{feature_name}}_service,
        }
    }
}
```

### 3. Wire Up in main.rs

```rust
use crate::features::{
    // ... existing imports
    {{feature_name}}_protected_routes,
    {{feature_name_pascal}}ServiceImpl,
};

// In main():

// Create service
let {{feature_name}}_service = Arc::new({{feature_name_pascal}}ServiceImpl::new(db.clone()));

// Add to AppState
let app_state = AppState::new(
    db,
    settings.clone(),
    task_service,
    category_service,
    auth_service,
    {{feature_name}}_service,  // Add here
);

// Add routes
let api_v1 = Router::new()
    .merge(health_routes())
    .merge(auth_routes())
    .merge(task_protected_routes(app_state.clone()))
    .merge(category_protected_routes(app_state.clone()))
    .merge({{feature_name}}_protected_routes(app_state.clone()));  // Add here
```

### 4. Add Database Schema

Create migration in `db/migrations/` (e.g., `005_{{feature_name}}.surql`):
```surql
-- Migration: 005_{{feature_name}}
-- Description: {{feature_name_pascal}} table and indexes

DEFINE TABLE {{table_name}} SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth != NONE
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

DEFINE FIELD name ON {{table_name}} TYPE string;
DEFINE FIELD description ON {{table_name}} TYPE option<string>;
DEFINE FIELD created_at ON {{table_name}} TYPE datetime VALUE time::now();
DEFINE FIELD updated_at ON {{table_name}} TYPE datetime VALUE time::now();
DEFINE FIELD deleted_at ON {{table_name}} TYPE option<datetime>;
DEFINE FIELD created_by ON {{table_name}} TYPE string;
DEFINE FIELD updated_by ON {{table_name}} TYPE string;

DEFINE INDEX idx_{{table_name}}_owner ON TABLE {{table_name}} COLUMNS created_by;
```

### 5. Update OpenAPI Documentation

In `main.rs`, add to the `ApiDoc` struct:
```rust
#[derive(OpenApi)]
#[openapi(
    paths(
        // ... existing paths
        features::{{feature_name}}::handler::list_{{feature_name}}s,
        features::{{feature_name}}::handler::get_{{feature_name}},
        features::{{feature_name}}::handler::create_{{feature_name}},
        features::{{feature_name}}::handler::update_{{feature_name}},
        features::{{feature_name}}::handler::delete_{{feature_name}},
    ),
    components(
        schemas(
            // ... existing schemas
            features::{{feature_name}}::model::{{feature_name_pascal}},
            features::{{feature_name}}::model::Create{{feature_name_pascal}}Request,
            features::{{feature_name}}::model::Update{{feature_name_pascal}}Request,
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
