# Rust Backend

A high-performance backend built with Axum, following industry best practices.

## Features

- **Axum Framework**: Fast, ergonomic web framework
- **SurrealDB**: Modern multi-model database
- **JWT Authentication**: Secure token-based auth
- **Auto-reload**: Development server with cargo-watch
- **Structured Logging**: tracing and tracing-subscriber
- **Validation**: Request validation with validator
- **Error Handling**: Comprehensive error types with thiserror

## Project Structure

```
src/
├── config/         # Configuration management
├── error/          # Error types and handling
├── handlers/       # HTTP request handlers
├── models/         # Data models and DTOs
├── repositories/   # Database access layer
├── utils/          # Utilities and middleware
└── main.rs         # Application entry point
```

## Setup

1. Copy the environment file:
```bash
cp .env.example .env
```

2. Install cargo-watch for auto-reload:
```bash
cargo install cargo-watch
```

3. Start the database (from project root):
```bash
task db:up
```

4. Run the development server:
```bash
cargo watch -x run
```

Or use the task command from project root:
```bash
task rust:dev
```

## API Endpoints

### Public
- `GET /health` - Health check
- `POST /auth/login` - User login
- `POST /auth/register` - User registration

### Protected (requires JWT)
- `GET /tasks` - List tasks (with pagination)
- `POST /tasks` - Create task
- `GET /tasks/:id` - Get task by ID
- `PUT /tasks/:id` - Update task
- `DELETE /tasks/:id` - Delete task

## Environment Variables

See `.env.example` for all available configuration options.

## Development

The server will automatically reload on file changes when using `cargo watch -x run`.

## Building for Production

```bash
cargo build --release
```

The optimized binary will be in `target/release/rust_backend`.
