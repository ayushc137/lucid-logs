//! Task repository with SurrealDB
//!
//! Uses SurrealDB's fluent builders, typed IDs, and schema-defined
//! functions to keep DB access safe and maintainable.
//!
//! # SurrealDB 2.x Patterns Used
//!
//! - **Fluent builders**: `db.create()`, `db.update().merge()`
//! - **Server-side functions**: `fn::task::count_for_user()`, `fn::task::with_category()`
//! - **Typed payloads**: Serde structs for insert/update content
//! - **Tuple keys**: `(table, id)` for update operations

use chrono::{DateTime, Utc};
use rand::Rng;
use serde::{Deserialize, Serialize};
use surrealdb::{engine::remote::ws::Client, RecordId, Surreal};

use super::model::{CreateTaskRequest, Task, UpdateTaskRequest, SOURCE_MANUAL};
use crate::core::error::AppError;
use crate::features::categories::Category;
use crate::shared::db::DbResultExt;
use crate::shared::{CategoryId, TaskId};

/// Server-side function to count tasks for a user (excludes soft-deleted)
const COUNT_TASKS_FN: &str = "RETURN fn::task::count_for_user($user)";

/// Raw query for paginated listing with FETCH (fluent builders don't support FETCH)
const LIST_BY_USER_QUERY: &str = r"
    SELECT * FROM tasks
    WHERE created_by = $user AND deleted_at = NONE
    ORDER BY start_date DESC
    LIMIT $limit START $offset
    FETCH category
";

/// Server-side function to get task with category populated
const FETCH_TASK_FUNCTION: &str = "RETURN fn::task::with_category(type::thing($id))";
const INSERT_TASK_QUERY: &str = r#"
    CREATE type::thing($id) CONTENT {
        title: $title,
        journal: $journal,
        start_date: type::datetime($start_date),
        end_date: type::datetime($end_date),
        completed: false,
        priority: $priority,
        source: $source,
        note: $note,
        positives: $positives,
        negatives: $negatives,
        category: {category_sql},
        created_by: $user,
        updated_by: $user,
        created_at: time::now(),
        updated_at: time::now(),
        deleted_at: NONE
    }
"#;

const UPDATE_TASK_QUERY: &str = r#"
    UPDATE type::thing($id) SET
        title = $title,
        journal = $journal,
        start_date = type::datetime($start_date),
        end_date = type::datetime($end_date),
        completed = $completed,
        priority = $priority,
        source = $source,
        note = $note,
        positives = $positives,
        negatives = $negatives,
        category = {category_sql},
        updated_by = $user,
        updated_at = time::now()
    WHERE created_by = $user AND deleted_at = NONE
    RETURN id
"#;
const CATEGORY_NOT_FOUND_MSG: &str = "Category not found or has been deleted";

/// Wrapper for extracting scalar count from server-side function
#[derive(Debug, Deserialize)]
struct CountWrapper(i64);


#[derive(Serialize)]
struct TaskSoftDeletePayload {
    deleted_at: Option<DateTime<Utc>>,
    updated_by: String,
    updated_at: DateTime<Utc>,
}

/// Task repository for database operations
///
/// Handles all task CRUD operations with ownership enforcement.
/// All queries filter by `created_by` to ensure data isolation.
#[derive(Clone)]
pub struct TaskRepository {
    db: Surreal<Client>,
}

impl TaskRepository {
    /// Create a new task repository instance
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new task with optional category link
    ///
    /// Uses SurrealDB's fluent `create().content()` builder for type-safe insertion,
    /// then calls server-side function to return the task with category populated.
    ///
    /// # Arguments
    /// - `req`: Task creation request with all required fields
    /// - `user_id`: ID of the user creating the task
    ///
    /// # Returns
    /// The newly created task with category populated
    pub async fn create(&self, req: CreateTaskRequest, user_id: &str) -> Result<Task, AppError> {
        let CreateTaskRequest {
            title,
            journal,
            start_date,
            end_date,
            priority,
            source,
            note,
            positives,
            negatives,
            category_id,
        } = req;

        let category_link = self
            .resolve_category_link(category_id, user_id)
            .await?;
        let task_id = generate_task_id();

        let insert_sql =
            INSERT_TASK_QUERY.replace("{category_sql}", &category_sql(&category_link));

        let result = self
            .db
            .query(&insert_sql)
            .bind(("id", task_id.full_id()))
            .bind(("title", title.clone()))
            .bind(("journal", journal))
            .bind(("start_date", start_date.time_value().to_rfc3339()))
            .bind(("end_date", end_date.time_value().to_rfc3339()))
            .bind(("priority", priority))
            .bind(("source", source.unwrap_or_else(|| SOURCE_MANUAL.to_string())))
            .bind(("note", note.unwrap_or_default()))
            .bind(("positives", positives.unwrap_or_default()))
            .bind(("negatives", negatives.unwrap_or_default()))
            .bind(("user", user_id.to_string()))
            .await;

        if let Err(ref err) = result {
            tracing::error!(
                ?err,
                %task_id,
                user_id,
                title = %title,
                "task insert failed"
            );
        }
        result?;

        self.fetch_task_with_category(&task_id, user_id).await
    }

    /// List tasks for a user with pagination
    ///
    /// Uses server-side function `fn::task::count_for_user()` for efficient counting
    /// and raw query for pagination with FETCH (fluent builders don't support FETCH).
    ///
    /// # Arguments
    /// - `user_id`: User ID for ownership filter
    /// - `limit`: Maximum number of tasks to return
    /// - `offset`: Number of tasks to skip
    ///
    /// # Returns
    /// Tuple of (tasks, total_count)
    pub async fn find_by_user_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError> {
        // Batch both queries in a single round-trip
        let mut result = self
            .db
            .query(COUNT_TASKS_FN)
            .query(LIST_BY_USER_QUERY)
            .bind(("user", user_id.to_string()))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        // Server-side function returns scalar directly
        let total: Option<CountWrapper> = result.take(0)?;
        let total = total.map(|w| w.0).unwrap_or(0);

        let mut tasks: Vec<Task> = result.take(1)?;
        tasks.iter_mut().for_each(sanitize_category);

        Ok((tasks, total))
    }

    /// Get a task by ID with category populated
    ///
    /// # Arguments
    /// - `id`: Task ID (with or without table prefix)
    /// - `user_id`: User ID for ownership verification
    ///
    /// # Returns
    /// The task if found and owned by user, NotFound otherwise
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Task, AppError> {
        let task_id = TaskId::new(id);
        self.fetch_task_with_category(&task_id, user_id).await
    }

    /// Update a task (enforces ownership)
    ///
    /// Only provided fields are updated; others remain unchanged.
    /// Ownership is verified before the update.
    ///
    /// # Arguments
    /// - `id`: Task ID to update
    /// - `req`: Update request with optional field values
    /// - `user_id`: User ID for ownership verification
    pub async fn update(
        &self,
        id: &str,
        req: UpdateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        let mut existing = self.find_by_id(id, user_id).await?;

        if let Some(title) = req.title {
            existing.title = title;
        }
        if let Some(journal) = req.journal {
            existing.journal = journal;
        }
        if let Some(start_date) = req.start_date {
            existing.start_date = start_date.time_value();
        }
        if let Some(end_date) = req.end_date {
            existing.end_date = end_date.time_value();
        }
        if let Some(completed) = req.completed {
            existing.completed = completed;
        }
        if let Some(priority) = req.priority {
            existing.priority = priority;
        }
        if let Some(note) = req.note {
            existing.note = note;
        }
        if let Some(positives) = req.positives {
            existing.positives = positives;
        }
        if let Some(negatives) = req.negatives {
            existing.negatives = negatives;
        }

        let category_link = match req.category_id {
            Some(ref raw) if raw.trim().is_empty() => Ok(None),
            Some(raw) => self.resolve_category_link(Some(raw), user_id).await,
            None => Ok(existing_category_link(&existing.category)),
        }?;

        let update_sql =
            UPDATE_TASK_QUERY.replace("{category_sql}", &category_sql(&category_link));

        let task_id = TaskId::new(id);
        let result = self
            .db
            .query(&update_sql)
            .bind(("id", task_id.full_id()))
            .bind(("title", existing.title))
            .bind(("journal", existing.journal))
            .bind(("start_date", existing.start_date.to_rfc3339()))
            .bind(("end_date", existing.end_date.to_rfc3339()))
            .bind(("completed", existing.completed))
            .bind(("priority", existing.priority))
            .bind(("source", existing.source))
            .bind(("note", existing.note))
            .bind(("positives", existing.positives))
            .bind(("negatives", existing.negatives))
            .bind(("user", user_id.to_string()))
            .await;

        if let Err(ref err) = result {
            tracing::error!(?err, %task_id, user_id, "task update failed");
        }

        let mut result = result?;
        let updated: Option<RecordId> = result.take(0)?;
        if updated.is_none() {
            return Err(AppError::NotFound);
        }

        self.find_by_id(id, user_id).await
    }

    /// Soft-delete a task
    ///
    /// Uses SurrealDB's fluent `update().merge()` to set `deleted_at` timestamp
    /// instead of actually removing the record.
    /// Ownership is verified as part of the update.
    ///
    /// # Arguments
    /// - `id`: Task ID to delete
    /// - `user_id`: User ID for ownership verification
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        // Ensure ownership before mutation
        let _ = self.find_by_id(id, user_id).await?;

        let task_id = TaskId::new(id);
        let now = Utc::now();
        let payload = TaskSoftDeletePayload {
            deleted_at: Some(now),
            updated_by: user_id.to_string(),
            updated_at: now,
        };

        self.db
            .update::<Option<Task>>(task_id.as_key())
            .merge(payload)
            .await
            .log_db_err(|err| {
                tracing::error!(?err, %task_id, user_id, "task delete failed");
            })?;

        tracing::info!(task_id = %task_id, "task soft-deleted");
        Ok(())
    }
}

fn existing_category_link(category: &Option<Category>) -> Option<String> {
    category
        .as_ref()
        .filter(|cat| cat.deleted_at.is_none())
        .and_then(|cat| cat.id.as_deref())
        .map(|id| CategoryId::new(id).full_id())
}

impl TaskRepository {
    async fn fetch_task_with_category(&self, id: &TaskId, user_id: &str) -> Result<Task, AppError> {
        let mut result = self
            .db
            .query(FETCH_TASK_FUNCTION)
            .bind(("id", id.full_id()))
            .await?;

        let tasks: Vec<Task> = result.take(0)?;
        let mut task = tasks.into_iter().next().ok_or_else(|| {
            tracing::warn!(task_id = %id, user_id, "task fetch returned empty result");
            AppError::NotFound
        })?;

        if task._created_by != user_id || task.deleted_at.is_some() {
            return Err(AppError::NotFound);
        }

        sanitize_category(&mut task);
        Ok(task)
    }

    async fn resolve_category_link(
        &self,
        category_id: Option<String>,
        user_id: &str,
    ) -> Result<Option<String>, AppError> {
        let Some(raw) = category_id
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty()) else {
            return Ok(None);
        };

        let category_id = CategoryId::new(raw);
        let record: Option<Category> = self.db.select(category_id.as_key()).await?;

        match record {
            Some(category) if category.deleted_at.is_none() && category._created_by == user_id => {
                Ok(Some(category_id.full_id()))
            },
            _ => {
                tracing::warn!(%category_id, user_id, "category validation failed");
                Err(AppError::BadRequest(CATEGORY_NOT_FOUND_MSG.into()))
            },
        }
    }
}

fn sanitize_category(task: &mut Task) {
    if let Some(category) = &task.category {
        if category.deleted_at.is_some() {
            task.category = None;
        }
    }
}

fn generate_task_id() -> TaskId {
    let mut rng = rand::rng();
    let value: u128 = rng.random();
    TaskId::new(format!("{value:032x}"))
}

fn category_sql(link: &Option<String>) -> String {
    match link {
        Some(id) => format!("type::thing('{}')", escape_quotes(id)),
        None => "NONE".to_string(),
    }
}

fn escape_quotes(value: &str) -> String {
    value.replace('\'', "''")
}
