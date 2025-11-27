# Rust for Beginners - Learning Through Our Codebase

Welcome! This guide explains Rust concepts **using actual code from this project**. If you're coming from JavaScript, Python, Go, or other languages, this will help you understand why Rust does things differently.

> 💡 **Tip**: Keep this guide open alongside the code. Each section links to real files in this project.

## Table of Contents

- [The Big Ideas](#the-big-ideas)
- [Ownership & Borrowing](#ownership--borrowing)
- [Smart Pointers (Arc, Box)](#smart-pointers-arc-box)
- [Traits (Like Interfaces)](#traits-like-interfaces)
- [Error Handling (Result & ?)](#error-handling-result--)
- [Async/Await](#asyncawait)
- [Derive Macros](#derive-macros)
- [Lifetimes](#lifetimes)
- [Pattern Matching](#pattern-matching)
- [Modules & Visibility](#modules--visibility)
- [Common Gotchas](#common-gotchas)
- [Learning Resources](#learning-resources)

---

## The Big Ideas

Rust has three core principles that differ from most languages:

1. **Ownership**: Every value has exactly one owner. When the owner goes out of scope, the value is dropped (freed).
2. **Borrowing**: You can temporarily lend values with references (`&` or `&mut`), but strict rules prevent data races.
3. **No Null**: Instead of null/nil, Rust uses `Option<T>` (Some or None) and `Result<T, E>` (Ok or Err).

These rules are enforced at **compile time**, which is why Rust catches bugs that other languages miss.

---

## Ownership & Borrowing

### The Problem Rust Solves

In other languages, you might have multiple variables pointing to the same data, leading to bugs:
- Use-after-free
- Double-free
- Data races in concurrent code

### How We Use It

📁 **See**: `src/main.rs` lines 166-173

```rust
// Create services with dependency injection
let task_service = Arc::new(TaskServiceImpl::new(db.clone()));
let auth_service = Arc::new(AuthServiceImpl::new(db.clone(), settings.clone()));

let app_state = AppState::new(db, settings.clone(), task_service, auth_service);
```

**What's happening here:**
- `db.clone()` - The database connection is cloned (actually just the reference count increases because it's wrapped in Arc)
- `settings.clone()` - Same for settings
- We pass ownership of `db` to `AppState::new()` - after this line, we can't use `db` directly anymore

### Key Syntax

| Syntax | Meaning |
|--------|---------|
| `let x = value` | `x` owns `value` |
| `&x` | Immutable borrow (read-only reference) |
| `&mut x` | Mutable borrow (can modify) |
| `x.clone()` | Create a deep copy (if Clone is implemented) |

### The Borrowing Rules

1. You can have **either** many `&T` OR one `&mut T` (not both)
2. References must always be valid (no dangling pointers)

📁 **See**: `src/handlers/task.rs` lines 67-91

```rust
pub async fn list_tasks(
    State(state): State<AppState>,        // state is owned by this function
    auth_user: AuthenticatedUser,          // auth_user is owned too
    Query(params): Query<PaginationParams>,
) -> Result<(StatusCode, Json<ApiResponse<PaginatedResponse<Task>>>), AppError> {
    // We use &auth_user.user_id - borrowing, not taking ownership
    let (tasks, total) = state
        .task_service
        .list_tasks(&auth_user.user_id, params.limit, params.offset)  // &str borrow
        .await?;
    // ...
}
```

---

## Smart Pointers (Arc, Box)

### Why We Need Them

Sometimes ownership rules are too restrictive:
- Multiple parts of your app need to share data
- You need runtime-determined sizes (like trait objects)

### Arc<T> - Atomic Reference Counting

📁 **See**: `src/utils/state.rs`

```rust
pub struct AppState {
    pub db: Surreal<Client>,
    pub settings: Arc<Settings>,                    // Shared settings
    pub task_service: Arc<dyn TaskService>,         // Shared service (trait object)
    pub auth_service: Arc<dyn AuthService>,
}
```

**Why Arc?**
- Multiple HTTP handlers run concurrently
- They all need access to the same services
- `Arc` lets them share safely (reference count + thread-safe)

**Arc vs Rc:**
- `Arc` = Atomic Reference Counting (thread-safe, slight overhead)
- `Rc` = Reference Counting (single-thread only, faster)
- In web servers, always use `Arc` because requests run on different threads

### dyn Trait - Dynamic Dispatch

```rust
Arc<dyn TaskService>
```

This means:
- "A pointer to *something* that implements TaskService"
- The concrete type is determined at runtime
- Enables dependency injection and mocking

📁 **See**: `tests/common/mod.rs` - `MockTaskService` implements the same trait, so we can swap it in for tests.

---

## Traits (Like Interfaces)

Traits define shared behavior, similar to interfaces in other languages.

📁 **See**: `src/services/traits.rs`

```rust
/// Task management service trait
#[async_trait]
pub trait TaskService: Send + Sync {
    /// List tasks for a user with pagination
    async fn list_tasks(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError>;

    /// Create a new task
    async fn create_task(&self, req: CreateTaskRequest, user_id: &str) -> Result<Task, AppError>;
    
    // ... more methods
}
```

### Key Concepts

| Concept | Explanation |
|---------|-------------|
| `#[async_trait]` | Macro that enables async methods in traits (Rust doesn't support this natively yet) |
| `Send + Sync` | Marker traits saying "safe to send between threads" and "safe to share between threads" |
| `&self` | Method takes a reference to the instance (like `this` in other languages) |

### Implementation

📁 **See**: `src/services/task.rs`

```rust
pub struct TaskServiceImpl {
    repo: TaskRepository,
}

#[async_trait]
impl TaskService for TaskServiceImpl {
    async fn list_tasks(&self, user_id: &str, limit: i64, offset: i64) -> Result<(Vec<Task>, i64), AppError> {
        // Actual implementation...
    }
}
```

### Why Traits Matter

1. **Abstraction**: Handlers depend on traits, not concrete types
2. **Testing**: Swap real services with mocks
3. **Flexibility**: Change implementations without changing consumers

---

## Error Handling (Result & ?)

Rust doesn't have exceptions. Instead, errors are values you must handle.

### Result<T, E>

```rust
enum Result<T, E> {
    Ok(T),    // Success with value of type T
    Err(E),   // Error with value of type E
}
```

### The ? Operator (Error Propagation)

📁 **See**: `src/main.rs` lines 140-155

```rust
// Sign in as root user for initial setup
db.signin(surrealdb::opt::auth::Root {
    username: &settings.db.user,
    password: &settings.db.pass,
})
.await?;  // <-- The ? operator

// Select namespace and database
db.use_ns(&settings.db.namespace)
    .use_db(&settings.db.database)
    .await?;
```

**What `?` does:**
1. If the Result is `Ok(value)`, unwrap and continue with `value`
2. If the Result is `Err(e)`, immediately return `Err(e)` from the current function

This is equivalent to:
```rust
// Without ?
let result = db.signin(...).await;
match result {
    Ok(value) => value,
    Err(e) => return Err(e.into()),
}
```

### Custom Error Types

📁 **See**: `src/error/mod.rs`

```rust
#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("Database error: {0}")]
    Database(#[from] surrealdb::Error),

    #[error("Validation error: {0}")]
    Validation(#[from] validator::ValidationErrors),

    #[error("Not found")]
    NotFound,

    #[error("Unauthorized: {0}")]
    Unauthorized(String),
    // ...
}
```

**Key parts:**
- `#[derive(thiserror::Error)]` - Auto-generates Error trait implementation
- `#[from]` - Auto-generates `From` trait for converting other error types
- `#[error("...")]` - Defines the error message format

---

## Async/Await

Rust's async is similar to JavaScript/Python but with ownership rules.

### How It Works

📁 **See**: `src/handlers/task.rs`

```rust
pub async fn create_task(
    State(state): State<AppState>,
    auth_user: AuthenticatedUser,
    Json(req): Json<CreateTaskRequest>,
) -> Result<(StatusCode, Json<ApiResponse<Task>>), AppError> {
    req.validate()?;

    let task = state.task_service
        .create_task(req, &auth_user.user_id)
        .await?;  // <-- Await the async operation

    Ok((StatusCode::CREATED, Json(ApiResponse::success(task))))
}
```

### Key Concepts

| Concept | Explanation |
|---------|-------------|
| `async fn` | Function returns a Future (like a Promise) |
| `.await` | Pause until the Future completes |
| `#[tokio::main]` | Macro that sets up the Tokio runtime for async |
| `tokio::spawn` | Run a Future in the background |

### Tokio Runtime

📁 **See**: `src/main.rs` line 81

```rust
#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // This function is now async!
}
```

The `#[tokio::main]` macro transforms this into:
```rust
fn main() -> anyhow::Result<()> {
    tokio::runtime::Runtime::new()
        .unwrap()
        .block_on(async { /* your code */ })
}
```

---

## Derive Macros

Derive macros auto-generate code. They're like decorators but run at compile time.

📁 **See**: `src/models/task.rs`

```rust
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct Task {
    pub id: Option<String>,
    pub title: String,
    // ...
}
```

### Common Derives

| Derive | What It Generates |
|--------|-------------------|
| `Debug` | `{:?}` formatting for debugging |
| `Clone` | `.clone()` method for copying |
| `Serialize` | Convert to JSON/other formats (serde) |
| `Deserialize` | Parse from JSON/other formats (serde) |
| `Default` | `Default::default()` with sensible defaults |
| `PartialEq` | `==` and `!=` comparisons |
| `ToSchema` | OpenAPI schema generation (utoipa) |
| `Validate` | Field validation (validator) |

### Attribute Macros

```rust
#[derive(Deserialize, Validate)]
pub struct CreateTaskRequest {
    #[validate(length(min = 1, max = 255))]  // Validation rules
    pub title: String,
    
    #[serde(default)]  // Use default if not provided
    pub priority: i32,
}
```

---

## Lifetimes

Lifetimes tell the compiler how long references are valid. You usually don't need to write them explicitly.

### When You See `'a`

```rust
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}
```

This says: "The returned reference lives as long as both input references."

### In Our Codebase

We mostly avoid explicit lifetimes by:
- Using `String` instead of `&str` in structs
- Using `Clone` when needed
- Using owned types in async code

📁 **See**: `src/models/task.rs` - all fields are owned types, no lifetimes needed.

### `'static` Lifetime

```rust
pub trait TaskService: Send + Sync + 'static { }
```

`'static` means the type contains no non-static references (can live forever if needed).

---

## Pattern Matching

Rust's `match` is more powerful than switch statements.

📁 **See**: `src/error/mod.rs` lines 117-184

```rust
impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, code, message, details) = match &self {
            AppError::Database(e) => {
                tracing::error!(error = %e, "database error");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "INTERNAL_ERROR",
                    "An internal error occurred".to_string(),
                    None,
                )
            }
            AppError::Validation(e) => {
                // Handle validation errors...
            }
            AppError::NotFound => (
                StatusCode::NOT_FOUND,
                "NOT_FOUND",
                "Resource not found".to_string(),
                None,
            ),
            // ... more patterns
        };
        // ...
    }
}
```

### Pattern Types

```rust
match value {
    Some(x) => /* x is the inner value */,
    None => /* handle missing */,
}

match result {
    Ok(data) => /* use data */,
    Err(AppError::NotFound) => /* specific error */,
    Err(e) => /* any other error */,
}

// Destructuring
let (first, second) = tuple;
let Point { x, y } = point;
```

---

## Modules & Visibility

📁 **See**: `src/main.rs` lines 11-17

```rust
mod config;      // Load src/config/mod.rs or src/config.rs
mod error;       // Load src/error/mod.rs
mod handlers;    // Load src/handlers/mod.rs
mod models;
mod repositories;
mod services;
mod utils;
```

### Visibility Rules

| Keyword | Visibility |
|---------|------------|
| (none) | Private to current module |
| `pub` | Public to everyone |
| `pub(crate)` | Public within this crate only |
| `pub(super)` | Public to parent module |

### Re-exports

📁 **See**: `src/services/mod.rs`

```rust
mod auth;
mod task;
pub mod traits;

pub use auth::AuthServiceImpl;
pub use task::TaskServiceImpl;
pub use traits::{AuthService, TaskService};
```

`pub use` re-exports items, so callers can do:
```rust
use crate::services::TaskService;  // Instead of crate::services::traits::TaskService
```

---

## Common Gotchas

### 1. String vs &str

| Type | What It Is | When to Use |
|------|------------|-------------|
| `String` | Owned, heap-allocated | When you need to store/modify |
| `&str` | Borrowed string slice | Function parameters, temporary use |

```rust
fn greet(name: &str) {        // Accept either String or &str
    println!("Hello, {}", name);
}

greet("literal");             // &str
greet(&my_string);            // &String coerces to &str
```

### 2. Move vs Copy

Primitive types (i32, f64, bool) are `Copy` - they're duplicated on assignment:
```rust
let x = 5;
let y = x;  // y is a copy, x still valid
```

Complex types (String, Vec, your structs) are moved:
```rust
let s1 = String::from("hello");
let s2 = s1;  // s1 is MOVED to s2
// println!("{}", s1);  // ERROR: s1 is no longer valid!
```

### 3. Turbofish `::<>`

When the compiler can't infer types:
```rust
let numbers: Vec<i32> = vec![1, 2, 3];
// Or with turbofish:
let numbers = vec![1, 2, 3].into_iter().collect::<Vec<i32>>();
```

### 4. Unused Variables

Prefix with `_` to silence warnings:
```rust
let _unused = compute_something();
```

---

## Learning Resources

### Official Resources
- [The Rust Book](https://doc.rust-lang.org/book/) - Start here
- [Rust by Example](https://doc.rust-lang.org/rust-by-example/) - Learn through examples
- [Rustlings](https://github.com/rust-lang/rustlings) - Small exercises

### This Project
- `cargo doc --open` - Browse our code's documentation
- `task rust:bacon` - Background compiler that shows errors as you type

### Getting Help
- Error messages: Rust's errors are actually helpful! Read them carefully.
- `cargo clippy` - Lints that suggest improvements
- [Rust Playground](https://play.rust-lang.org/) - Test snippets online

### Cheat Sheets
- [Rust Language Cheat Sheet](https://cheats.rs/)
- See `docs/RUST_CHEATSHEET.md` in this project

---

## Quick Exercises

Try these to practice:

1. **Add a field**: Add a `tags: Vec<String>` field to `Task` and update the handlers
2. **New endpoint**: Add `GET /api/v1/tasks/:id` endpoint
3. **Error variant**: Add a new error type to `AppError`
4. **Mock test**: Write a test using `MockTaskService`

---

> 💡 **Remember**: The compiler is your friend. If it compiles, it (usually) works correctly!

