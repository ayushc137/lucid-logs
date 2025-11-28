# Code Walkthrough: Request Journey

This document traces a complete HTTP request through our codebase, explaining each step. We'll follow a `POST /api/v1/tasks` request to create a new task.

## Overview

```
HTTP Request
    │
    ▼
┌─────────────┐
│   Router    │  ← Route matching & middleware
└─────────────┘
    │
    ▼
┌─────────────┐
│  Handler    │  ← Request parsing & validation
└─────────────┘
    │
    ▼
┌─────────────┐
│  Service    │  ← Business logic
└─────────────┘
    │
    ▼
┌─────────────┐
│ Repository  │  ← Database operations
└─────────────┘
    │
    ▼
┌─────────────┐
│  Database   │  ← SurrealDB
└─────────────┘
```

---

## Step 1: Application Bootstrap

📁 **File**: `src/main.rs`

When the server starts, we set up everything:

### 1.1 Load Configuration

```rust
let _ = dotenvy::dotenv();           // Load .env file
let settings = Settings::new()?;     // Parse into typed config
```

### 1.2 Connect to Database

```rust
let db: Surreal<Client> = match tokio::time::timeout(
    std::time::Duration::from_secs(10),
    Surreal::new::<Ws>(&db_url),
).await { ... };
```

### 1.3 Create Services (Dependency Injection)

```rust
let task_service = Arc::new(TaskServiceImpl::new(db.clone()));
let category_service = Arc::new(CategoryServiceImpl::new(db.clone()));
let auth_service = Arc::new(AuthServiceImpl::new(db.clone(), settings.clone()));

let app_state = AppState::new(
    db,
    settings.clone(),
    task_service,
    category_service,
    auth_service,
);
```

**Why Arc?** Multiple request handlers need to access these services concurrently. `Arc` provides thread-safe shared ownership.

### 1.4 Build Router

```rust
let api_v1 = Router::new()
    .merge(health_routes())
    .merge(auth_routes())
    .merge(task_protected_routes(app_state.clone()))
    .merge(category_protected_routes(app_state.clone()));

let app = Router::new()
    .route("/health", get(features::health::health_check))
    .nest("/api/v1", api_v1)
    .with_state(app_state)
    .layer(TraceLayer::new_for_http())
    .layer(cors);
```

---

## Step 2: Request Arrives

A client sends:

```http
POST /api/v1/tasks HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Content-Type: application/json

{
  "title": "Learn Rust",
  "journal": "Programming",
  "start_date": "2024-01-15T09:00:00Z",
  "end_date": "2024-01-15T10:00:00Z",
  "priority": 1
}
```

---

## Step 3: Router Matches the Route

📁 **File**: `src/features/tasks/handler.rs`

```rust
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/tasks", get(list_tasks))
        .route("/tasks", post(create_task))    // ← This matches!
        .route("/tasks/{id}", get(get_task))
        .route("/tasks/{id}", put(update_task))
        .route("/tasks/{id}", delete(delete_task))
}
```

The path `/api/v1/tasks` with method `POST` routes to `create_task`.

---

## Step 4: Middleware Runs First

📁 **File**: `src/core/middleware.rs`

Before the handler, auth middleware validates the JWT:

```rust
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request<Body>,
    next: Next,
) -> Response {
    // 1. Extract Authorization header
    let auth_header = req.headers()
        .get(header::AUTHORIZATION)
        .and_then(|h| h.to_str().ok())
        .map(|s| s.trim().to_string());

    let auth_header = match auth_header {
        Some(h) if !h.is_empty() => h,
        _ => return unauthorized_response("Authorization header required"),
    };

    // 2. Parse "Bearer <token>"
    const BEARER_PREFIX: &str = "Bearer ";
    if !auth_header.starts_with(BEARER_PREFIX) {
        return unauthorized_response("Authorization header must start with 'Bearer '");
    }

    let token = auth_header[BEARER_PREFIX.len()..].trim();

    // 3. Verify JWT and extract claims using SurrealClaims
    let claims = match SurrealClaims::from_token(token, &state.settings.jwt.secret) {
        Ok(claims) => claims,
        Err(e) => return unauthorized_response("Invalid or expired token"),
    };

    // 4. Inject authenticated user into request extensions
    req.extensions_mut().insert(AuthenticatedUser { user_id: claims.id });

    // 5. Continue to the handler
    next.run(req).await
}
```

If authentication fails, an error response is returned immediately.

---

## Step 5: Handler Receives Request

📁 **File**: `src/features/tasks/handler.rs`

```rust
#[utoipa::path(...)]  // OpenAPI documentation
pub async fn create_task(
    State(state): State<AppState>,              // Extract app state
    auth_user: AuthenticatedUser,               // Extract from middleware
    Json(req): Json<CreateTaskRequest>,         // Parse JSON body
) -> Result<(StatusCode, Json<ApiResponse<Task>>), AppError> {
```

### Axum Extractors Explained

| Extractor | What It Does |
|-----------|--------------|
| `State(state)` | Extracts cloned `AppState` from router |
| `AuthenticatedUser` | Custom extractor (from request extensions) |
| `Json(req)` | Parses request body as JSON into `CreateTaskRequest` |
| `Path(id)` | Extracts URL path parameters |
| `Query(params)` | Extracts URL query parameters |

### 5.1 Validate Request

```rust
req.validate()?;
```

📁 **File**: `src/features/tasks/model.rs`

```rust
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateTaskRequest {
    #[validate(length(min = 1, message = "Title is required"))]
    pub title: String,

    #[serde(default)]
    pub journal: String,

    pub start_date: DateTimeInput,
    pub end_date: DateTimeInput,
    // ...
}
```

If validation fails, `?` returns early with `AppError::Validation`.

### 5.2 Call Service

```rust
let task = state.task_service
    .create_task(req, &auth_user.user_id)
    .await?;
```

The handler delegates business logic to the service layer.

---

## Step 6: Service Processes Business Logic

📁 **File**: `src/features/tasks/service.rs`

```rust
#[async_trait]
impl TaskService for TaskServiceImpl {
    async fn create_task(
        &self,
        req: CreateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        // Business logic validation
        if req.end_date.time_value() < req.start_date.time_value() {
            return Err(AppError::BadRequest(
                "end_date must be on or after start_date".to_string()
            ));
        }

        // Delegate to repository
        self.repo.create(req, user_id).await
    }
}
```

**Service responsibilities:**
- Business rule validation
- Orchestrating multiple repository calls
- Transforming data between layers

---

## Step 7: Repository Interacts with Database

📁 **File**: `src/features/tasks/repository.rs`

```rust
impl TaskRepository {
    pub async fn create(
        &self,
        req: CreateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        // Build INSERT query with type-safe bindings
        let create_sql = format!(
            r#"
            INSERT INTO tasks {{
                title: $title,
                journal: $journal,
                start_date: type::datetime($start_date),
                end_date: type::datetime($end_date),
                completed: false,
                priority: $priority,
                source: $source,
                category: {category_sql},
                created_by: $user,
                updated_by: $user,
                created_at: time::now(),
                updated_at: time::now()
            }} RETURN id
            "#
        );

        let mut result = self.db
            .query(&create_sql)
            .bind(("title", req.title))
            .bind(("journal", req.journal))
            .bind(("start_date", req.start_date.time_value().to_rfc3339()))
            .bind(("end_date", req.end_date.time_value().to_rfc3339()))
            .bind(("priority", req.priority))
            .bind(("user", user_id.to_string()))
            .await?;

        // Extract created task ID and fetch full record
        let created: Option<IdResult> = result.take(0)?;
        let task_id = match created {
            Some(r) => TaskId::new(r.id_string()),
            None => return Err(AppError::Internal),
        };

        // Return complete task with category populated
        self.find_by_id(&task_id.full_id(), user_id).await
    }
}
```

**Repository responsibilities:**
- Database queries using centralized query registry
- Type-safe record IDs (`TaskId`, `CategoryId`)
- Data mapping (DB format ↔ Domain models)
- No business logic!

---

## Step 8: Response Flows Back

### 8.1 Repository Returns Task

```rust
Ok(Task { id: Some("tasks:abc123"), title: "Learn Rust", ... })
```

### 8.2 Service Returns Task

Same `Task` passes through (no transformation needed here).

### 8.3 Handler Wraps in ApiResponse

```rust
// Lines 129
Ok((StatusCode::CREATED, Json(ApiResponse::success(task))))
```

📁 **File**: `src/core/error.rs`

```rust
impl<T: Serialize + ToSchema> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            data: Some(data),
            error: None,
        }
    }
}
```

### 8.4 Axum Serializes to JSON

Final response:

```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "data": {
    "id": "tasks:abc123",
    "title": "Learn Rust",
    "journal": "Programming",
    "start_date": "2024-01-15T09:00:00Z",
    "end_date": "2024-01-15T10:00:00Z",
    "completed": false,
    "priority": 1,
    "created_at": "2024-01-15T08:30:00Z",
    "updated_at": "2024-01-15T08:30:00Z",
    "created_by": "user:xyz789",
    "updated_by": "user:xyz789"
  },
  "error": null
}
```

---

## Error Flow

What happens when something goes wrong?

### Example: Task Not Found (DELETE /api/v1/tasks/invalid-id)

```
Handler
  │
  ├─► service.delete_task("invalid-id", user_id)
  │         │
  │         └─► repo.delete("invalid-id", user_id)
  │                   │
  │                   └─► returns Err(AppError::NotFound)
  │         │
  │         └─► ? operator propagates Err
  │
  └─► ? operator propagates Err to Axum
            │
            ▼
      AppError::into_response()
```

📁 **File**: `src/core/error.rs`

```rust
AppError::NotFound => (
    StatusCode::NOT_FOUND,
    "NOT_FOUND",
    "Resource not found".to_string(),
    None,
),
```

Response:

```http
HTTP/1.1 404 Not Found
Content-Type: application/json

{
  "data": null,
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found"
  }
}
```

---

## Key Files Summary

| File | Purpose |
|------|---------|
| `src/main.rs` | App bootstrap, router setup |
| `src/state.rs` | Shared application state (AppState) |
| `src/core/config.rs` | Configuration management |
| `src/core/error.rs` | Error types and API response wrappers |
| `src/core/middleware.rs` | Auth middleware |
| `src/core/db/migrations.rs` | Database migration runner |
| `src/features/*/handler.rs` | HTTP layer (parsing, responses) |
| `src/features/*/service.rs` | Business logic |
| `src/features/*/repository.rs` | Database operations |
| `src/features/*/model.rs` | Data structures (DTOs, entities) |
| `src/shared/db/types.rs` | Type-safe record IDs |
| `src/shared/db/queries.rs` | Centralized SQL query registry |

---

## Exercises

1. **Trace a GET request**: Follow `GET /api/v1/tasks?limit=10` through the code
2. **Add logging**: Add `tracing::debug!` statements to see the flow
3. **Break something**: Remove a `?` and see what error the compiler gives
4. **Add a field**: Add `description` to Task and trace all files that need updating

