use chrono::Utc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::base::{ensure_record_id, CountResult};
use crate::error::AppError;
use crate::models::task::{CreateTaskRequest, Task, UpdateTaskRequest, SOURCE_MANUAL};

/// Task projection fields for queries (matching Go implementation)
const TASK_PROJECTION: &str = "id, title, journal, start_date, end_date, is_completed, priority, planned, source, note, positives, negatives, created_at, updated_at, deleted_at, created_by, updated_by";

/// Table name for tasks (must match schema.surql)
const TASKS_TABLE: &str = "tasks";

#[derive(Clone)]
pub struct TaskRepository {
    db: Surreal<Client>,
}

impl TaskRepository {
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new task
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
        } = req;

        let start_date = start_date.time_value();
        let end_date = end_date.time_value();

        // Calculate planned status
        let now = Utc::now();
        let planned = (start_date >= now) && (start_date != end_date);

        let source = source.unwrap_or_else(|| SOURCE_MANUAL.to_string());
        let note = note.unwrap_or_default();
        let positives = positives.unwrap_or_default();
        let negatives = negatives.unwrap_or_default();
        let start_iso = start_date.to_rfc3339();
        let end_iso = end_date.to_rfc3339();
        let user_owned = user_id.to_string();

        let sql = format!(
            "CREATE {} SET
                title = $title,
                journal = $journal,
                start_date = type::datetime($start_date),
                end_date = type::datetime($end_date),
                is_completed = false,
                priority = $priority,
                planned = $planned,
                source = $source,
                note = $note,
                positives = $positives,
                negatives = $negatives,
                created_by = $user,
                updated_by = $user
             RETURN {}",
            TASKS_TABLE, TASK_PROJECTION
        );

        tracing::debug!(title = %title, "creating task in database");

        let mut result = self
            .db
            .query(sql)
            .bind(("title", title))
            .bind(("journal", journal))
            .bind(("start_date", start_iso))
            .bind(("end_date", end_iso))
            .bind(("priority", priority))
            .bind(("planned", planned))
            .bind(("source", source))
            .bind(("note", note))
            .bind(("positives", positives))
            .bind(("negatives", negatives))
            .bind(("user", user_owned))
            .await?;

        let created: Option<Task> = result.take(0)?;

        match created {
            Some(task) => {
                tracing::debug!(task_id = ?task.id, "task created successfully");
                Ok(task)
            }
            None => {
                // Log any errors from the query response
                let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
                if !errors.is_empty() {
                    tracing::error!(?errors, "database create failed with errors");
                } else {
                    tracing::error!("database create failed - no task returned and no errors reported");
                }
                Err(AppError::Internal)
            }
        }
    }

    /// List tasks for a user with pagination (excludes soft-deleted)
    /// Returns (tasks, total_count)
    pub async fn find_by_user_paginated(
        &self,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError> {
        let user_id_owned = user_id.to_string();

        // Get count and items in a single query batch
        let count_sql = format!(
            "SELECT count() FROM {} WHERE created_by = $user AND deleted_at = NONE GROUP ALL",
            TASKS_TABLE
        );
        let items_sql = format!(
            "SELECT {} FROM {} WHERE created_by = $user AND deleted_at = NONE ORDER BY start_date DESC LIMIT $limit START $offset",
            TASK_PROJECTION, TASKS_TABLE
        );

        let mut result = self
            .db
            .query(count_sql)
            .query(items_sql)
            .bind(("user", user_id_owned))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        // Extract count from first query
        let count_result: Option<CountResult> = result.take(0)?;
        let total = count_result.map(|c| c.count).unwrap_or(0);

        // Extract tasks from second query
        let tasks: Vec<Task> = result.take(1)?;

        Ok((tasks, total))
    }

    /// Get a task by ID (excludes soft-deleted, enforces ownership)
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Task, AppError> {
        let record_id = ensure_record_id(id, TASKS_TABLE);
        let sql = format!(
            "SELECT {} FROM {} WHERE id = type::thing($record) AND created_by = $user AND deleted_at = NONE",
            TASK_PROJECTION, TASKS_TABLE
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("user", user_id.to_string()))
            .await?;

        let tasks: Vec<Task> = result.take(0)?;
        // Return NotFound for both non-existent and unauthorized (prevents ID enumeration)
        tasks.into_iter().next().ok_or(AppError::NotFound)
    }

    /// Update a task (enforces ownership)
    pub async fn update(
        &self,
        id: &str,
        req: UpdateTaskRequest,
        user_id: &str,
    ) -> Result<Task, AppError> {
        // Fetch existing task (ownership verified by find_by_id)
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
        if let Some(is_completed) = req.is_completed {
            existing.is_completed = is_completed;
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

        // Recalculate planned status
        let now = Utc::now();
        existing.planned = (existing.start_date >= now) && (existing.start_date != existing.end_date);

        let record_id = ensure_record_id(id, TASKS_TABLE);
        // Note: created_by is NOT updated - preserves original owner
        let sql = format!(
            "UPDATE type::thing($record) SET
                title = $title,
                journal = $journal,
                start_date = type::datetime($start_date),
                end_date = type::datetime($end_date),
                is_completed = $is_completed,
                priority = $priority,
                planned = $planned,
                source = $source,
                note = $note,
                positives = $positives,
                negatives = $negatives,
                updated_by = $updated_by
             WHERE created_by = $user
             RETURN {}",
            TASK_PROJECTION
        );

        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("title", existing.title))
            .bind(("journal", existing.journal))
            .bind(("start_date", existing.start_date.to_rfc3339()))
            .bind(("end_date", existing.end_date.to_rfc3339()))
            .bind(("is_completed", existing.is_completed))
            .bind(("priority", existing.priority))
            .bind(("planned", existing.planned))
            .bind(("source", existing.source))
            .bind(("note", existing.note))
            .bind(("positives", existing.positives))
            .bind(("negatives", existing.negatives))
            .bind(("updated_by", user_id.to_string()))
            .bind(("user", user_id.to_string()))
            .await?;

        let updated: Option<Task> = result.take(0)?;
        updated.ok_or(AppError::NotFound)
    }

    /// Soft-delete a task (sets deleted_at timestamp, enforces ownership)
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let record_id = ensure_record_id(id, TASKS_TABLE);
        let user_id_owned = user_id.to_string();
        let now = Utc::now().to_rfc3339();

        // Add ownership check to prevent deleting others' tasks
        let sql = format!(
            "UPDATE type::thing($record) SET deleted_at = type::datetime($now), updated_by = $user WHERE created_by = $user AND deleted_at = NONE RETURN {}",
            TASK_PROJECTION
        );
        let mut result = self
            .db
            .query(sql)
            .bind(("record", record_id))
            .bind(("now", now))
            .bind(("user", user_id_owned))
            .await?;

        let deleted: Option<Task> = result.take(0)?;
        if deleted.is_none() {
            // Return NotFound for both non-existent and unauthorized (prevents ID enumeration)
            return Err(AppError::NotFound);
        }

        tracing::info!(task_id = %id, "task soft-deleted successfully");
        Ok(())
    }
}

