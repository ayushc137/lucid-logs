# Rust Cheatsheet for This Project

Quick reference for common patterns used in this codebase.

---

## Creating Things

### New Struct Instance
```rust
// With all fields
let task = Task {
    id: Some("tasks:123".to_string()),
    title: "My Task".to_string(),
    // ... all fields
};

// With Default for remaining fields
let config = Config {
    port: 8080,
    ..Default::default()
};
```

### New Handler
```rust
use axum::{extract::State, Json};
use crate::{error::{ApiResponse, AppError}, utils::state::AppState};

pub async fn my_handler(
    State(state): State<AppState>,
) -> Result<Json<ApiResponse<MyResponse>>, AppError> {
    let result = do_something(&state).await?;
    Ok(Json(ApiResponse::success(result)))
}
```

### New Service Trait Method
```rust
// In services/traits.rs
#[async_trait]
pub trait MyService: Send + Sync {
    async fn do_thing(&self, input: Input) -> Result<Output, AppError>;
}

// In services/my_service.rs
#[async_trait]
impl MyService for MyServiceImpl {
    async fn do_thing(&self, input: Input) -> Result<Output, AppError> {
        // implementation
    }
}
```

---

## String Operations

```rust
// Create String
let s = String::from("hello");
let s = "hello".to_string();
let s = format!("hello {}", name);

// String to &str
let slice: &str = &my_string;

// &str to String
let owned: String = my_str.to_string();
let owned: String = my_str.into();

// Concatenation
let s = format!("{}{}", s1, s2);
let s = [s1, s2].concat();

// Check contents
if s.is_empty() { }
if s.contains("needle") { }
if s.starts_with("prefix") { }
```

---

## Option & Result

### Option<T>
```rust
let opt: Option<i32> = Some(42);
let opt: Option<i32> = None;

// Unwrap (panics if None - avoid in production!)
let val = opt.unwrap();

// Safe unwrap with default
let val = opt.unwrap_or(0);
let val = opt.unwrap_or_default();

// Transform
let doubled = opt.map(|x| x * 2);

// Chain
let result = opt
    .filter(|x| x > &10)
    .map(|x| x.to_string());

// Pattern match
match opt {
    Some(val) => println!("Got {}", val),
    None => println!("Nothing"),
}

// If let (when you only care about Some)
if let Some(val) = opt {
    println!("Got {}", val);
}
```

### Result<T, E>
```rust
let result: Result<i32, String> = Ok(42);
let result: Result<i32, String> = Err("failed".into());

// Propagate error with ?
let val = risky_operation()?;  // Returns early if Err

// Handle error
match result {
    Ok(val) => println!("Success: {}", val),
    Err(e) => println!("Error: {}", e),
}

// Transform
let doubled = result.map(|x| x * 2);
let mapped_err = result.map_err(|e| AppError::Internal);

// Unwrap with default
let val = result.unwrap_or(0);

// Convert Option to Result
let result = opt.ok_or(AppError::NotFound)?;
```

---

## Collections

### Vec<T>
```rust
// Create
let v: Vec<i32> = Vec::new();
let v = vec![1, 2, 3];

// Add/Remove
v.push(4);
let last = v.pop();

// Access
let first = &v[0];           // Panics if out of bounds
let first = v.get(0);        // Returns Option<&T>
let first = v.first();       // Returns Option<&T>

// Iterate
for item in &v { }           // Borrow
for item in &mut v { }       // Mutable borrow
for item in v { }            // Takes ownership

// Transform
let doubled: Vec<i32> = v.iter().map(|x| x * 2).collect();
let filtered: Vec<_> = v.iter().filter(|x| **x > 1).collect();

// Find
let found = v.iter().find(|x| **x == 2);
let pos = v.iter().position(|x| *x == 2);
```

### HashMap<K, V>
```rust
use std::collections::HashMap;

// Create
let mut map: HashMap<String, i32> = HashMap::new();

// Insert
map.insert("key".to_string(), 42);

// Get
let val = map.get("key");           // Option<&V>
let val = map.get("key").copied();  // Option<V> for Copy types

// Entry API (insert if not exists)
map.entry("key".to_string()).or_insert(0);

// Iterate
for (key, value) in &map { }
```

---

## Async Patterns

### Basic Async
```rust
async fn fetch_data() -> Result<Data, AppError> {
    let response = client.get(url).await?;
    let data = response.json().await?;
    Ok(data)
}
```

### Concurrent Execution
```rust
use tokio::join;

// Run concurrently, wait for all
let (result1, result2) = join!(
    async_fn1(),
    async_fn2(),
);

// Run concurrently, wait for first
use tokio::select;
select! {
    result = async_fn1() => { /* first finished */ }
    result = async_fn2() => { /* second finished */ }
}
```

### Spawn Background Task
```rust
tokio::spawn(async move {
    // This runs in background
    do_something().await;
});
```

---

## Error Handling Patterns

### Early Return with ?
```rust
async fn process() -> Result<Output, AppError> {
    let data = fetch().await?;        // Returns if Err
    let validated = validate(data)?;   // Returns if Err
    let result = transform(validated);
    Ok(result)
}
```

### Custom Error Conversion
```rust
// In error/mod.rs, add:
#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("My new error: {0}")]
    MyError(String),
    
    // Auto-convert from another error type
    #[error("External error: {0}")]
    External(#[from] external_crate::Error),
}
```

### Map Errors
```rust
let result = risky_fn()
    .map_err(|e| AppError::BadRequest(e.to_string()))?;
```

---

## Struct Patterns

### Builder Pattern (with `bon`)
```rust
use bon::Builder;

#[derive(Builder)]
pub struct Config {
    pub host: String,
    pub port: u16,
    #[builder(default)]
    pub debug: bool,
}

// Usage
let config = Config::builder()
    .host("localhost".to_string())
    .port(8080)
    .build();
```

### Newtype Pattern
```rust
// Wrap a type for type safety
pub struct UserId(String);

impl UserId {
    pub fn new(id: impl Into<String>) -> Self {
        Self(id.into())
    }
    
    pub fn as_str(&self) -> &str {
        &self.0
    }
}
```

---

## Testing Patterns

### Unit Test
```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_something() {
        let result = my_function(input);
        assert_eq!(result, expected);
    }
}
```

### Async Test
```rust
#[tokio::test]
async fn test_async_fn() {
    let result = async_function().await;
    assert!(result.is_ok());
}
```

### With Mock Service
```rust
use crate::tests::common::MockTaskService;

#[tokio::test]
async fn test_with_mock() {
    let mock = MockTaskService::with_tasks(vec![/* test data */]);
    let result = mock.list_tasks("user:1", 10, 0).await;
    assert_eq!(result.unwrap().0.len(), expected_count);
}
```

### Parameterized Tests (rstest)
```rust
use rstest::rstest;

#[rstest]
#[case(1, 2, 3)]
#[case(2, 2, 4)]
#[case(0, 5, 5)]
fn test_add(#[case] a: i32, #[case] b: i32, #[case] expected: i32) {
    assert_eq!(a + b, expected);
}
```

### Snapshot Tests (insta)
```rust
use insta::assert_json_snapshot;

#[test]
fn test_response_format() {
    let response = create_response();
    assert_json_snapshot!(response);
}
```

---

## Logging (tracing)

```rust
use tracing::{debug, info, warn, error, instrument};

// Simple logging
info!("Server started");
debug!(port = 8080, "Listening");
warn!(user_id = %id, "Rate limit approaching");
error!(error = %e, "Database connection failed");

// Structured fields
info!(
    user_id = %user.id,
    action = "login",
    "User logged in"
);

// Instrument functions (auto-log entry/exit)
#[instrument(skip(password))]  // Don't log password!
async fn login(username: &str, password: &str) -> Result<Token, Error> {
    // ...
}
```

---

## Common Axum Extractors

```rust
use axum::extract::{State, Path, Query, Json};

// All extractors in one handler
async fn handler(
    State(state): State<AppState>,           // App state
    Path(id): Path<String>,                  // URL path param
    Query(params): Query<PaginationParams>,  // Query string
    Json(body): Json<CreateRequest>,         // JSON body
    auth: AuthenticatedUser,                 // Custom extractor
) -> Result<Json<Response>, AppError> {
    // ...
}
```

---

## Serde Attributes

```rust
#[derive(Serialize, Deserialize)]
pub struct MyStruct {
    // Rename for JSON
    #[serde(rename = "userName")]
    pub user_name: String,
    
    // Skip if None
    #[serde(skip_serializing_if = "Option::is_none")]
    pub optional_field: Option<String>,
    
    // Default value
    #[serde(default)]
    pub count: i32,
    
    // Custom default
    #[serde(default = "default_priority")]
    pub priority: i32,
    
    // Flatten nested struct
    #[serde(flatten)]
    pub metadata: Metadata,
}

fn default_priority() -> i32 {
    1
}
```

---

## Quick Reference

### Type Conversions
```rust
// String conversions
let s: String = "text".to_string();
let s: String = "text".into();
let s: &str = &string;

// Number conversions
let n: i64 = i32_val.into();
let n: i32 = i64_val as i32;  // May truncate!

// Option<T> to Result<T, E>
let result = option.ok_or(AppError::NotFound)?;
```

### Common Traits to Derive
```rust
#[derive(
    Debug,        // {:?} formatting
    Clone,        // .clone() method
    Default,      // Default::default()
    PartialEq,    // == comparison
    Serialize,    // To JSON
    Deserialize,  // From JSON
)]
```

### Import Aliases
```rust
use std::collections::HashMap as Map;
use crate::error::AppError as Error;
```

