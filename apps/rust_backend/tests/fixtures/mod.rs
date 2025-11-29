//! Test fixtures and sample data
//!
//! This module provides static test data and fixture files.

/// Sample valid JWT token for testing (expired, not for production use)
pub const SAMPLE_JWT: &str = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJJRCI6InVzZXI6dGVzdCIsIk5TIjoidGVzdCIsIkRCIjoidGVzdCIsIkFDIjoiYWNjb3VudCIsImlhdCI6MTcwMDAwMDAwMCwiZXhwIjoxOTAwMDAwMDAwfQ.test";

/// Sample user ID for testing
pub const TEST_USER_ID: &str = "user:test123";

/// Sample task JSON for testing
pub const SAMPLE_TASK_JSON: &str = r#"{
    "title": "Test Task",
    "journal": "Test journal entry",
    "start_date": "2025-01-01T09:00:00Z",
    "end_date": "2025-01-01T17:00:00Z",
    "priority": 1
}"#;

/// Sample invalid task JSON (missing required fields)
pub const INVALID_TASK_JSON: &str = r#"{
    "journal": "Missing title"
}"#;

/// Sample auth request JSON
pub const SAMPLE_AUTH_JSON: &str = r#"{
    "username": "test@example.com",
    "password": "testpassword123"
}"#;
