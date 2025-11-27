//! Task repository with SurrealDB graph relations
//!
//! Uses graph traversal for category relationships:
//! - task -[in_category]-> category
//!
//! Query pattern: `->in_category->categories AS category`

use chrono::Utc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use super::base::{ensure_record_id, CountResult};
use crate::error::AppError;
use crate::models::task::{CreateTaskRequest, Task, UpdateTaskRequest, SOURCE_MANUAL};

/// Task projection fields (without category - that comes from graph traversal)
const TASK_PROJECTION: &str = "id, title, journal, start_date, end_date, completed, priority, source, note, positives, negatives, created_at, updated_at, deleted_at, created_by, updated_by";

/// Graph traversal to get category
const CATEGORY_TRAVERSAL: &str = "->in_category->categories[0] AS category";

/// Table names
const TASKS_TABLE: &str = "tasks";
const IN_CATEGORY_EDGE: &str = "in_category";

#[derive(Clone)]
pub struct TaskRepository {
    db: Surreal<Client>,
}

impl TaskRepository {
    pub fn new(db: Surreal<Client>) -> Self {
        Self { db }
    }

    /// Create a new task with optional category relation
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

        // Create task
        let create_sql = format!(
            "CREATE {} SET
                title = $title,
                journal = $journal,
                start_date = type::datetime($start_date),
                end_date = type::datetime($end_date),
                completed = false,
                priority = $priority,
                source = $source,
                note = $note,
                positives = $positives,
                negatives = $negatives,
                created_by = $user,
                updated_by = $user
             RETURN id",
            TASKS_TABLE
        );

        tracing::debug!(title = %title, ?category_id, "creating task with graph relation");

        let mut result = self
            .db
            .query(&create_sql)
            .bind(("title", title))
            .bind(("journal", journal))
            .bind(("start_date", start_iso))
            .bind(("end_date", end_iso))
            .bind(("priority", priority))
            .bind(("source", source))
            .bind(("note", note))
            .bind(("positives", positives))
            .bind(("negatives", negatives))
            .bind(("user", user_owned.clone()))
            .await?;

        #[derive(serde::Deserialize)]
        struct IdResult {
            id: surrealdb::RecordId,
        }

        let created: Option<IdResult> = result.take(0)?;
        let task_id = match created {
            Some(r) => r.id.to_string(),
            None => {
                let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
                if !errors.is_empty() {
                    tracing::error!(?errors, "database create failed with errors");
                }
                return Err(AppError::Internal);
            }
        };

        // Create category relation if provided
        if let Some(cat_id) = category_id {
            if !cat_id.is_empty() {
                let cat_record = ensure_record_id(&cat_id, "categories");
                self.set_category_relation(&task_id, &cat_record, &user_owned)
                    .await?;
            }
        }

        // Fetch and return the complete task with category
        self.find_by_id(&task_id, user_id).await
    }

    /// Set or replace category relation for a task
    async fn set_category_relation(
        &self,
        task_id: &str,
        category_id: &str,
        user_id: &str,
    ) -> Result<(), AppError> {
        let task_record = ensure_record_id(task_id, TASKS_TABLE);

        // Delete existing relation (if any) and create new one
        // Using transaction-like batch query
        let sql = format!(
            "DELETE {} WHERE in = type::thing($task);
             RELATE type::thing($task) -> {} -> type::thing($category) 
               SET assigned_by = $user;",
            IN_CATEGORY_EDGE, IN_CATEGORY_EDGE
        );

        self.db
            .query(&sql)
            .bind(("task", task_record))
            .bind(("category", category_id.to_string()))
            .bind(("user", user_id.to_string()))
            .await?;

        Ok(())
    }

    /// Remove category relation from a task
    async fn remove_category_relation(&self, task_id: &str) -> Result<(), AppError> {
        let task_record = ensure_record_id(task_id, TASKS_TABLE);

        let sql = format!("DELETE {} WHERE in = type::thing($task)", IN_CATEGORY_EDGE);

        self.db
            .query(&sql)
            .bind(("task", task_record))
            .await?;

        Ok(())
    }

    /// List tasks for a user with pagination (excludes soft-deleted)
    /// Categories are populated via graph traversal
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

        // Items query with graph traversal for category
        let items_sql = format!(
            "SELECT {}, {} FROM {} WHERE created_by = $user AND deleted_at = NONE ORDER BY start_date DESC LIMIT $limit START $offset",
            TASK_PROJECTION, CATEGORY_TRAVERSAL, TASKS_TABLE
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

    /// Get a task by ID with category populated via graph traversal
    pub async fn find_by_id(&self, id: &str, user_id: &str) -> Result<Task, AppError> {
        let record_id = ensure_record_id(id, TASKS_TABLE);

        let sql = format!(
            "SELECT {}, {} FROM {} WHERE id = type::thing($record) AND created_by = $user AND deleted_at = NONE",
            TASK_PROJECTION, CATEGORY_TRAVERSAL, TASKS_TABLE
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

        // Update task fields
        let sql = "UPDATE type::thing($record) SET
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
                updated_by = $updated_by
             WHERE created_by = $user
             RETURN id";

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
            #[allow(dead_code)]
            id: surrealdb::RecordId,
        }

        let updated: Option<IdResult> = result.take(0)?;
        if updated.is_none() {
            return Err(AppError::NotFound);
        }

        // Handle category relation update
        if let Some(cat_id) = req.category_id {
            if cat_id.is_empty() {
                // Empty string = remove category
                self.remove_category_relation(id).await?;
            } else {
                // Set new category
                let cat_record = ensure_record_id(&cat_id, "categories");
                self.set_category_relation(id, &cat_record, user_id).await?;
            }
        }
        // If category_id is None, keep existing relation unchanged

        // Fetch and return updated task with category
        self.find_by_id(id, user_id).await
    }

    /// Soft-delete a task and its relations
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let record_id = ensure_record_id(id, TASKS_TABLE);
        let user_id_owned = user_id.to_string();
        let now = Utc::now().to_rfc3339();

        // Soft delete the task
        let sql = format!(
            "UPDATE type::thing($record) SET deleted_at = type::datetime($now), updated_by = $user WHERE created_by = $user AND deleted_at = NONE RETURN id",
        );

        let mut result = self
            .db
            .query(&sql)
            .bind(("record", record_id.clone()))
            .bind(("now", now))
            .bind(("user", user_id_owned))
            .await?;

        #[derive(serde::Deserialize)]
        struct IdResult {
            #[allow(dead_code)]
            id: surrealdb::RecordId,
        }

        let deleted: Option<IdResult> = result.take(0)?;
        if deleted.is_none() {
            return Err(AppError::NotFound);
        }

        // Also remove any category relations for this task
        self.remove_category_relation(id).await?;

        tracing::info!(task_id = %id, "task and relations soft-deleted");
        Ok(())
    }

    /// Find all tasks in a specific category (graph query example)
    #[allow(dead_code)]
    pub async fn find_by_category(
        &self,
        category_id: &str,
        user_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<(Vec<Task>, i64), AppError> {
        let cat_record = ensure_record_id(category_id, "categories");

        // Use reverse graph traversal: category <-in_category<- tasks
        let count_sql = format!(
            "SELECT count() FROM {} WHERE id IN (SELECT in FROM {} WHERE out = type::thing($category)) AND created_by = $user AND deleted_at = NONE GROUP ALL",
            TASKS_TABLE, IN_CATEGORY_EDGE
        );

        let items_sql = format!(
            "SELECT {}, {} FROM {} WHERE id IN (SELECT in FROM {} WHERE out = type::thing($category)) AND created_by = $user AND deleted_at = NONE ORDER BY start_date DESC LIMIT $limit START $offset",
            TASK_PROJECTION, CATEGORY_TRAVERSAL, TASKS_TABLE, IN_CATEGORY_EDGE
        );

        let mut result = self
            .db
            .query(count_sql)
            .query(items_sql)
            .bind(("category", cat_record))
            .bind(("user", user_id.to_string()))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        let count_result: Option<CountResult> = result.take(0)?;
        let total = count_result.map(|c| c.count).unwrap_or(0);

        let tasks: Vec<Task> = result.take(1)?;

        Ok((tasks, total))
    }
}
