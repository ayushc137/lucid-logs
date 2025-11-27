//! Task repository with SurrealDB record links
//!
//! Uses direct record links for category relationships:
//! - task.category = categories:work123
//!
//! Query pattern: `SELECT *, category.* FROM tasks FETCH category`

use chrono::Utc;
use surrealdb::{engine::remote::ws::Client, Surreal};

use super::base::{ensure_record_id, CountResult};
use crate::error::AppError;
use crate::models::task::{CreateTaskRequest, Task, UpdateTaskRequest, SOURCE_MANUAL};

/// Table name
const TASKS_TABLE: &str = "tasks";

#[derive(Clone)]
pub struct TaskRepository {
    db: Surreal<Client>,
}

impl TaskRepository {
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new task with optional category link
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

        let start_date = start_date.time_value();
        let end_date = end_date.time_value();

        let source = source.unwrap_or_else(|| SOURCE_MANUAL.to_string());
        let note = note.unwrap_or_default();
        let positives = positives.unwrap_or_default();
        let negatives = negatives.unwrap_or_default();
        let start_iso = start_date.to_rfc3339();
        let end_iso = end_date.to_rfc3339();
        let user_owned = user_id.to_string();

        // Build category value - either a record link or NONE
        let category_value = match &category_id {
            Some(cat_id) if !cat_id.is_empty() => {
                let cat_record = ensure_record_id(cat_id, "categories");
                format!("type::thing('{}')", cat_record)
            },
            _ => "NONE".to_string(),
        };

        // Schemaless INSERT - just store whatever fields we want
        let create_sql = format!(
            "INSERT INTO {} {{
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
                category: {},
                created_by: $user,
                updated_by: $user,
                created_at: time::now(),
                updated_at: time::now(),
                deleted_at: NONE
            }} RETURN id",
            TASKS_TABLE, category_value
        );

        tracing::debug!(title = %title, ?category_id, "creating task with record link");

        let mut result = self
            .db
            .query(&create_sql)
            .bind(("title", title.clone()))
            .bind(("journal", journal))
            .bind(("start_date", start_iso))
            .bind(("end_date", end_iso))
            .bind(("priority", priority))
            .bind(("source", source))
            .bind(("note", note))
            .bind(("positives", positives))
            .bind(("negatives", negatives))
            .bind(("user", user_owned))
            .await?;

        // Check for errors
        let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
        if !errors.is_empty() {
            tracing::error!(?errors, "database create failed with errors");
            return Err(AppError::Internal);
        }

        #[derive(serde::Deserialize)]
        struct IdResult {
            id: surrealdb::RecordId,
        }

        let created: Option<IdResult> = result.take(0)?;
        let task_id = match created {
            Some(r) => r.id.to_string(),
            None => {
                tracing::error!(
                    title = %title,
                    "database create failed - no task returned and no errors reported"
                );
                return Err(AppError::Internal);
            },
        };

        // Fetch and return the complete task with category
        self.find_by_id(&task_id, user_id).await
    }

    /// List tasks for a user with pagination (excludes soft-deleted)
    /// Categories are populated via FETCH
    pub async fn find_by_user_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError> {
        let user_id_owned = user_id.to_string();

        // Count query
        let count_sql = format!(
            "SELECT count() FROM {} WHERE created_by = $user AND deleted_at = NONE GROUP ALL",
            TASKS_TABLE
        );

        // Items query with FETCH to hydrate category
        let items_sql = format!(
            "SELECT * FROM {} WHERE created_by = $user AND deleted_at = NONE ORDER BY start_date DESC LIMIT $limit START $offset FETCH category",
            TASKS_TABLE
        );

        let mut result = self
            .db
            .query(count_sql)
            .query(items_sql)
            .bind(("user", user_id_owned))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        let count_result: Option<CountResult> = result.take(0)?;
        let total = count_result.map(|c| c.count).unwrap_or(0);

        let tasks: Vec<Task> = result.take(1)?;

        Ok((tasks, total))
    }

    /// Get a task by ID with category populated via FETCH
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Task, AppError> {
        let record_id = ensure_record_id(id, TASKS_TABLE);

        let sql = format!(
            "SELECT * FROM {} WHERE id = type::thing($record) AND created_by = $user AND deleted_at = NONE FETCH category",
            TASKS_TABLE
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("user", user_id.to_string()))
            .await?;

        let tasks: Vec<Task> = result.take(0)?;
        tasks.into_iter().next().ok_or(AppError::NotFound)
    }

    /// Update a task (enforces ownership)
    pub async fn update(
        &self,
        id: &str,
        req: UpdateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        // Fetch existing task (ownership verified)
        let mut existing = self.find_by_id(id, user_id).await?;

        // Update fields if provided
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

        let record_id = ensure_record_id(id, TASKS_TABLE);

        // Build category update clause
        let category_clause = match &req.category_id {
            Some(cat_id) if cat_id.is_empty() => {
                // Empty string = remove category
                "category = NONE,".to_string()
            },
            Some(cat_id) => {
                // Set new category
                let cat_record = ensure_record_id(cat_id, "categories");
                format!("category = type::thing('{}'),", cat_record)
            },
            None => {
                // No change to category
                String::new()
            },
        };

        // Update task fields
        let sql = format!(
            "UPDATE type::thing($record) SET
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
                {}
                updated_by = $updated_by,
                updated_at = time::now()
             WHERE created_by = $user
             RETURN id",
            category_clause
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
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
            .bind(("updated_by", user_id.to_string()))
            .bind(("user", user_id.to_string()))
            .await?;

        #[derive(serde::Deserialize)]
        struct IdResult {
            #[serde(rename = "id")]
            _id: surrealdb::RecordId,
        }

        let updated: Option<IdResult> = result.take(0)?;
        if updated.is_none() {
            return Err(AppError::NotFound);
        }

        // Fetch and return updated task with category
        self.find_by_id(id, user_id).await
    }

    /// Soft-delete a task
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let record_id = ensure_record_id(id, TASKS_TABLE);
        let user_id_owned = user_id.to_string();
        let now = Utc::now().to_rfc3339();

        // Soft delete the task
        let sql =
            "UPDATE type::thing($record) SET deleted_at = type::datetime($now), updated_by = $user, updated_at = time::now() WHERE created_by = $user AND deleted_at = NONE RETURN id";

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("now", now))
            .bind(("user", user_id_owned))
            .await?;

        #[derive(serde::Deserialize)]
        struct IdResult {
            #[serde(rename = "id")]
            _id: surrealdb::RecordId,
        }

        let deleted: Option<IdResult> = result.take(0)?;
        if deleted.is_none() {
            return Err(AppError::NotFound);
        }

        tracing::info!(task_id = %id, "task soft-deleted");
        Ok(())
    }
}
