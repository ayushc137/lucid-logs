//! Task repository with SurrealDB
//!
//! Uses the centralized query registry and type-safe ID wrappers
//! for maintainable and safe database operations.
//!
//! # Architecture
//!
//! - All queries are defined in `shared::db::queries::tasks`
//! - Record IDs use type-safe `TaskId` and `CategoryId` wrappers
//! - Category relationships use SurrealDB record links with FETCH
//!
//! # Query Pattern
//!
//! ```sql
//! SELECT *, category.* FROM tasks FETCH category
//! ```
//!
//! This hydrates the linked category record in a single query.

use chrono::Utc;
use surrealdb::{engine::remote::ws::Client, Surreal};

use super::model::{CreateTaskRequest, Task, UpdateTaskRequest, SOURCE_MANUAL};
use crate::core::error::AppError;
use crate::shared::db::{CountResult, IdResult};
use crate::shared::{task_queries as queries, CategoryId, TaskId};

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

        // Prepare field values
        let source = source.unwrap_or_else(|| SOURCE_MANUAL.to_string());
        let note = note.unwrap_or_default();
        let positives = positives.unwrap_or_default();
        let negatives = negatives.unwrap_or_default();

        // Convert category_id to proper format for binding
        // SurrealDB needs either a Thing or NONE for record links
        let category_sql = match &category_id {
            Some(cat_id) if !cat_id.is_empty() => {
                let cat = CategoryId::new(cat_id);
                format!("type::thing('{}')", cat.full_id())
            },
            _ => "NONE".to_string(),
        };

        // Build CREATE query with inline category expression
        // We can't bind NONE directly, so we use string interpolation for this one field
        let create_sql = format!(
            r"
            INSERT INTO tasks {{
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
            }} RETURN id
            "
        );

        tracing::debug!(title = %title, ?category_id, sql = %create_sql, "creating task");

        let mut result = self
            .db
            .query(&create_sql)
            .bind(("title", title.clone()))
            .bind(("journal", journal))
            .bind(("start_date", start_date.time_value().to_rfc3339()))
            .bind(("end_date", end_date.time_value().to_rfc3339()))
            .bind(("priority", priority))
            .bind(("source", source))
            .bind(("note", note))
            .bind(("positives", positives))
            .bind(("negatives", negatives))
            .bind(("user", user_id.to_string()))
            .await?;

        // Check for query errors
        let errors: Vec<surrealdb::Error> = result.take_errors().into_values().collect();
        if !errors.is_empty() {
            tracing::error!(?errors, "task create failed with DB errors");
            return Err(AppError::Internal);
        }

        // Extract created task ID - try to take as Vec first to see all results
        let all_results: Vec<IdResult> = result.take(0)?;
        tracing::debug!(?all_results, "task create raw results");
        
        let task_id = match all_results.into_iter().next() {
            Some(r) => TaskId::new(r.id_string()),
            None => {
                tracing::error!(title = %title, "task create returned no result");
                return Err(AppError::Internal);
            },
        };

        // Fetch complete task with category populated
        self.find_by_id(&task_id.full_id(), user_id).await
    }

    /// List tasks for a user with pagination
    ///
    /// Returns tasks ordered by start_date descending.
    /// Soft-deleted tasks are excluded.
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
        // Execute count and list queries in a single batch
        let mut result = self
            .db
            .query(queries::COUNT_BY_USER)
            .query(queries::LIST_BY_USER)
            .bind(("user", user_id.to_string()))
            .bind(("limit", limit))
            .bind(("offset", offset))
            .await?;

        // Extract count from first query
        let count_result: Option<CountResult> = result.take(0)?;
        let total = CountResult::unwrap_or_zero(count_result);

        // Extract tasks from second query
        let tasks: Vec<Task> = result.take(1)?;

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

        let mut result = self
            .db
            .query(queries::SELECT_BY_ID)
            .bind(("id", task_id.full_id()))
            .bind(("user", user_id.to_string()))
            .await?;

        let tasks: Vec<Task> = result.take(0)?;
        tasks.into_iter().next().ok_or(AppError::NotFound)
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
        // Fetch existing task (verifies ownership and existence)
        let mut existing = self.find_by_id(id, user_id).await?;

        // Apply updates to existing record
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

        let task_id = TaskId::new(id);

        // Build category SQL fragment
        let category_sql = match &req.category_id {
            Some(cat_id) if cat_id.is_empty() => "NONE".to_string(),
            Some(cat_id) => {
                let cat = CategoryId::new(cat_id);
                format!("type::thing('{}')", cat.full_id())
            },
            None => {
                // Keep existing category - extract from current task
                match &existing.category {
                    Some(cat) => {
                        if let Some(cat_id) = &cat.id {
                            format!("type::thing('{}')", cat_id)
                        } else {
                            "NONE".to_string()
                        }
                    },
                    None => "NONE".to_string(),
                }
            },
        };

        // Build UPDATE query with category expression
        let update_sql = format!(
            r"
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
            "
        );

        let mut result = self
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
            .await?;

        let updated: Option<IdResult> = result.take(0)?;
        if updated.is_none() {
            return Err(AppError::NotFound);
        }

        // Fetch and return updated task with category
        self.find_by_id(id, user_id).await
    }

    /// Soft-delete a task
    ///
    /// Sets `deleted_at` timestamp instead of actually removing the record.
    /// Ownership is verified as part of the update.
    ///
    /// # Arguments
    /// - `id`: Task ID to delete
    /// - `user_id`: User ID for ownership verification
    pub async fn delete(&self, id: &str, user_id: &str) -> Result<(), AppError> {
        let task_id = TaskId::new(id);
        let now = Utc::now().to_rfc3339();

        let mut result = self
            .db
            .query(queries::SOFT_DELETE)
            .bind(("id", task_id.full_id()))
            .bind(("now", now))
            .bind(("user", user_id.to_string()))
            .await?;

        let deleted: Option<IdResult> = result.take(0)?;
        if deleted.is_none() {
            return Err(AppError::NotFound);
        }

        tracing::info!(task_id = %task_id, "task soft-deleted");
        Ok(())
    }
}

