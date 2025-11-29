//! Unit tests for type-safe record ID wrappers

#![allow(clippy::unwrap_used)]

use rust_backend::{CategoryId, TaskId};

#[test]
fn test_task_id_new() {
    let id = TaskId::new("abc123");
    assert_eq!(id.id(), "abc123");
    assert_eq!(id.full_id(), "tasks:abc123");
}

#[test]
fn test_task_id_strips_prefix() {
    let id = TaskId::new("tasks:abc123");
    assert_eq!(id.id(), "abc123");
    assert_eq!(id.full_id(), "tasks:abc123");
}

#[test]
fn test_task_id_parse() {
    let id = TaskId::parse("tasks:abc123").unwrap();
    assert_eq!(id.id(), "abc123");

    // Wrong table returns None
    assert!(TaskId::parse("categories:abc123").is_none());
}

#[test]
fn test_category_id_new() {
    let id = CategoryId::new("work");
    assert_eq!(id.full_id(), "categories:work");
}

#[test]
fn test_category_id_strips_prefix() {
    let id = CategoryId::new("categories:personal");
    assert_eq!(id.full_id(), "categories:personal");
}

#[test]
fn test_category_id_parse() {
    let id = CategoryId::parse("categories:work").unwrap();
    assert_eq!(id.id(), "work");

    // Wrong table returns None
    assert!(CategoryId::parse("tasks:work").is_none());
}
