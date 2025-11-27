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
// Line 84-87
let _ = dotenvy::dotenv();           // Load .env file
let settings = Settings::new()?;     // Parse into typed config
```

### 1.2 Connect to Database

```rust
// Lines 117-138
let db: Surreal<Client> = match tokio::time::timeout(
    std::time::Duration::from_secs(10),
    Surreal::new::<Ws>(&db_url),
).await { ... };
```

### 1.3 Create Services (Dependency Injection)

```rust
// Lines 169-173
let task_service = Arc::new(TaskServiceImpl::new(db.clone()));
let auth_service = Arc::new(AuthServiceImpl::new(db.clone(), settings.clone()));
let app_state = AppState::new(db, settings.clone(), task_service, auth_service);
```

**Why Arc?** Multiple request handlers need to access these services concurrently. `Arc` provides thread-safe shared ownership.

### 1.4 Build Router

```rust
// Lines 192-213
let api_v1 = Router::new()
    .merge(handlers::health::routes())
    .merge(handlers::auth::routes())
    .merge(handlers::task::protected_routes(app_state.clone()));

let app = Router::new()
    .route("/health", get(handlers::health::health_check))
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

📁 **File**: `src/handlers/task.rs`

```rust
// Lines 21-27
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/tasks", get(list_tasks))
        .route("/tasks", post(create_task))    // ← This matches!
        .route("/tasks/{id}", put(update_task))
        .route("/tasks/{id}", delete(delete_task))
}
```

The path `/api/v1/tasks` with method `POST` routes to `create_task`.

---

## Step 4: Middleware Runs First

📁 **File**: `src/utils/middleware.rs`

Before the handler, auth middleware validates the JWT:

```rust
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    // 1. Extract Authorization header
    let auth_header = req.headers()
        .get(AUTHORIZATION)
        .and_then(|h| h.to_str().ok())
        .ok_or_else(|| AppError::Unauthorized("Missing authorization header".into()))?;

    // 2. Parse "Bearer <token>"
    let token = auth_header
        .strip_prefix("Bearer ")
        .ok_or_else(|| AppError::Unauthorized("Invalid authorization format".into()))?;

    // 3. Verify JWT and extract claims
    let claims = verify_jwt(token, &state.settings.jwt.secret)?;

    // 4. Inject authenticated user into request extensions
    req.extensions_mut().insert(AuthenticatedUser {
        user_id: claims.sub,
    });

    // 5. Continue to the handler
    Ok(next.run(req).await)
}
```

If authentication fails, an error response is returned immediately.

---

## Step 5: Handler Receives Request

📁 **File**: `src/handlers/task.rs` (lines 107-130)

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
// Line 113
req.validate()?;
```

📁 **File**: `src/models/task.rs`

```rust
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateTaskRequest {
    #[validate(length(min = 1, max = 255, message = "title must be 1-255 characters"))]
    pub title: String,

    #[validate(length(min = 1, max = 100, message = "journal must be 1-100 characters"))]
    pub journal: String,
    // ...
}
```

If validation fails, `?` returns early with `AppError::Validation`.

### 5.2 Call Service

```rust
// Lines 122-123
let task = state.task_service
    .create_task(req, &auth_user.user_id)
    .await?;
```

The handler delegates business logic to the service layer.

---

## Step 6: Service Processes Business Logic

📁 **File**: `src/services/task.rs`

```rust
#[async_trait]
impl TaskService for TaskServiceImpl {
    async fn create_task(
        &self,
        req: CreateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        // Business logic validation
        if req.start_date.0 >= req.end_date.0 {
            return Err(AppError::BadRequest(
                "start_date must be before end_date".into()
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

📁 **File**: `src/repositories/task.rs`

```rust
impl TaskRepository {
    pub async fn create(
        &self,
        req: CreateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        let now = Utc::now();
        
        // Build the task record
        let task_data = serde_json::json!({
            "title": req.title,
            "journal": req.journal,
            "start_date": req.start_date.0,
            "end_date": req.end_date.0,
            "is_completed": false,
            "priority": req.priority,
            "created_at": now,
            "updated_at": now,
            "created_by": user_id,
            "updated_by": user_id,
        });

        // Insert into SurrealDB
        let result: Option<Task> = self.db
            .create(TABLE)
            .content(task_data)
            .await?;

        result.ok_or(AppError::Internal)
    }
}
```

**Repository responsibilities:**
- Database queries
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

📁 **File**: `src/error/mod.rs`

```rust
impl<T: Serialize> ApiResponse<T> {
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
    "is_completed": false,
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

📁 **File**: `src/error/mod.rs` (lines 152-157)

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
| `src/handlers/*.rs` | HTTP layer (parsing, responses) |
| `src/services/*.rs` | Business logic |
| `src/repositories/*.rs` | Database operations |
| `src/models/*.rs` | Data structures (DTOs, entities) |
| `src/error/mod.rs` | Error types and API response wrappers |
| `src/utils/middleware.rs` | Auth middleware |
| `src/utils/state.rs` | Shared application state |

---

## Exercises

1. **Trace a GET request**: Follow `GET /api/v1/tasks?limit=10` through the code
2. **Add logging**: Add `tracing::debug!` statements to see the flow
3. **Break something**: Remove a `?` and see what error the compiler gives
4. **Add a field**: Add `description` to Task and trace all files that need updating

