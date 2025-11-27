# Troubleshooting Guide

Common errors and how to fix them.

---

## Compilation Errors

### "borrow of moved value"

```
error[E0382]: borrow of moved value: `data`
  --> src/main.rs:10:20
   |
8  |     let data = get_data();
   |         ---- move occurs because `data` has type `String`
9  |     process(data);
   |             ---- value moved here
10 |     println!("{}", data);
   |                    ^^^^ value borrowed here after move
```

**Problem**: You used a value after giving ownership away.

**Solutions**:
```rust
// Option 1: Clone before moving
let data = get_data();
process(data.clone());
println!("{}", data);  // Works!

// Option 2: Pass a reference instead
let data = get_data();
process(&data);        // Borrow instead of move
println!("{}", data);  // Works!

// Option 3: Restructure to not need the value after
let data = get_data();
println!("{}", data);  // Use first
process(data);         // Then move
```

---

### "cannot borrow as mutable"

```
error[E0596]: cannot borrow `*self.data` as mutable, as it is behind a `&` reference
```

**Problem**: You're trying to mutate through an immutable reference.

**Solutions**:
```rust
// Option 1: Change function signature to &mut self
impl MyStruct {
    fn modify(&mut self) {  // Was &self
        self.data.push(item);
    }
}

// Option 2: Use interior mutability (Mutex, RwLock, RefCell)
use std::sync::Mutex;

struct MyStruct {
    data: Mutex<Vec<Item>>,
}

impl MyStruct {
    fn modify(&self) {
        self.data.lock().unwrap().push(item);
    }
}
```

---

### "lifetime may not live long enough"

```
error: lifetime may not live long enough
  --> src/lib.rs:5:5
   |
4  |     fn get_name(&self) -> &str {
   |                 - let's call the lifetime of this reference `'1`
5  |         &self.name
   |         ^^^^^^^^^^ returning this value requires that `'1` must outlive `'static`
```

**Problem**: Rust can't prove the returned reference lives long enough.

**Solutions**:
```rust
// Option 1: Return owned data instead
fn get_name(&self) -> String {
    self.name.clone()
}

// Option 2: Explicit lifetime annotations
fn get_name<'a>(&'a self) -> &'a str {
    &self.name
}

// Option 3: Use 'static if data truly is static
fn get_name(&self) -> &'static str {
    "constant"  // Only works for compile-time constants
}
```

---

### "expected X, found Y" (type mismatch)

```
error[E0308]: mismatched types
  --> src/main.rs:5:10
   |
5  |     foo(x)
   |         ^ expected `String`, found `&str`
```

**Solutions**:
```rust
// &str to String
foo(x.to_string())
foo(x.into())
foo(String::from(x))

// String to &str
foo(&x)
foo(x.as_str())

// Option<T> to T
foo(x.unwrap())         // Panics if None!
foo(x.unwrap_or(default))
foo(x?)                 // In function returning Result/Option

// &T to T (if T: Clone)
foo(x.clone())
foo((*x).clone())
```

---

### "the trait bound X is not satisfied"

```
error[E0277]: the trait bound `MyType: Serialize` is not satisfied
```

**Problem**: A type doesn't implement a required trait.

**Solutions**:
```rust
// Option 1: Derive the trait
#[derive(Serialize, Deserialize)]
struct MyType { ... }

// Option 2: Implement manually
impl Serialize for MyType {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        // ...
    }
}

// Option 3: Check if you're missing a feature flag in Cargo.toml
[dependencies]
chrono = { version = "0.4", features = ["serde"] }  # Add serde feature
```

---

### "async fn is not Send"

```
error: future cannot be sent between threads safely
   |
   = help: within `impl Future`, the trait `Send` is not implemented for `Rc<String>`
```

**Problem**: You're using non-thread-safe types in async code.

**Solutions**:
```rust
// Replace Rc with Arc
use std::sync::Arc;
let shared = Arc::new(data);

// Replace RefCell with Mutex/RwLock
use std::sync::Mutex;
let mutable = Mutex::new(data);

// Or use parking_lot for faster locks
use parking_lot::Mutex;
```

---

## Runtime Errors

### "called `Option::unwrap()` on a `None` value"

**Problem**: You used `.unwrap()` on `None`.

**Solutions**:
```rust
// Check first
if let Some(value) = optional {
    use(value);
}

// Provide default
let value = optional.unwrap_or(default);
let value = optional.unwrap_or_default();

// Return error instead of panic
let value = optional.ok_or(AppError::NotFound)?;
```

---

### "called `Result::unwrap()` on an `Err` value"

**Problem**: You used `.unwrap()` on an `Err`.

**Solutions**:
```rust
// Propagate error
let value = risky_operation()?;

// Handle error
match risky_operation() {
    Ok(value) => use(value),
    Err(e) => handle_error(e),
}

// Log and provide default
let value = risky_operation().unwrap_or_else(|e| {
    tracing::warn!(error = %e, "Operation failed, using default");
    default
});
```

---

### Database Connection Errors

```
Failed to connect to SurrealDB: Connection refused
```

**Solutions**:
1. Check if SurrealDB is running: `task db:up`
2. Check environment variables in `.env`
3. Verify port isn't blocked: `lsof -i :8000`
4. Check Docker logs: `docker logs surrealdb`

---

### "JWT decode error" / "Invalid token"

**Solutions**:
1. Check `JWT_SECRET` matches between token creation and verification
2. Verify token hasn't expired
3. Ensure token format is `Bearer <token>` in Authorization header
4. Check for whitespace in token

---

## Cargo/Build Issues

### "failed to select a version for X"

**Problem**: Dependency version conflict.

**Solutions**:
```bash
# Update Cargo.lock
cargo update

# Check what's conflicting
cargo tree -d

# Force a specific version (in Cargo.toml)
[dependencies]
problematic-crate = "=1.2.3"
```

---

### "can't find crate"

```
error[E0463]: can't find crate for `some_crate`
```

**Solutions**:
```bash
# Make sure it's in Cargo.toml
cargo add some_crate

# Rebuild
cargo clean
cargo build
```

---

### Build is slow

**Solutions**:
```bash
# Use mold linker (Linux)
# Add to .cargo/config.toml:
# [target.x86_64-unknown-linux-gnu]
# linker = "clang"
# rustflags = ["-C", "link-arg=-fuse-ld=mold"]

# Use sccache
cargo install sccache
export RUSTC_WRAPPER=sccache

# Use incremental compilation (default in dev)
# Make sure CARGO_INCREMENTAL isn't set to 0
```

---

## IDE Issues

### rust-analyzer not working

**Solutions**:
1. Restart rust-analyzer: `Cmd/Ctrl+Shift+P` → "rust-analyzer: Restart server"
2. Check `rust-analyzer.cargo.target` in settings
3. Delete `target/` and rebuild: `rm -rf target && cargo build`
4. Update rust-analyzer extension

---

### "proc macro not expanded"

**Problem**: Macros like `#[derive(...)]` show errors.

**Solutions**:
1. Enable proc-macro support in rust-analyzer settings:
```json
{
  "rust-analyzer.procMacro.enable": true
}
```
2. Rebuild: `cargo clean && cargo build`

---

## Common Mistakes

### Forgetting `.await`

```rust
// Wrong - returns Future, not result
let data = fetch_data();

// Correct
let data = fetch_data().await;
```

### Forgetting `?` for error propagation

```rust
// Wrong - ignores potential error
let data = risky_operation().await;

// Correct - propagates error
let data = risky_operation().await?;
```

### Using `==` instead of `.eq()` for Option

```rust
// Works
if option == Some(value) { }

// But for checking None, prefer:
if option.is_none() { }
if option.is_some() { }
```

### Mutating while iterating

```rust
// Wrong - can't modify while borrowing
for item in &mut vec {
    if should_remove(item) {
        vec.remove(i);  // Error!
    }
}

// Correct - use retain
vec.retain(|item| !should_remove(item));
```

---

## Getting More Help

### Read the full error

Rust's errors are detailed. Scroll up to see the full message and suggestions.

### Use clippy suggestions

```bash
cargo clippy --all-targets --all-features
```

Clippy often suggests better alternatives.

### Check documentation

```bash
# Open docs for a crate
cargo doc --open --package surrealdb

# Search docs.rs
# https://docs.rs/axum
```

### Ask for help

- [Rust Users Forum](https://users.rust-lang.org/)
- [Rust Discord](https://discord.gg/rust-lang)
- [Stack Overflow [rust] tag](https://stackoverflow.com/questions/tagged/rust)

