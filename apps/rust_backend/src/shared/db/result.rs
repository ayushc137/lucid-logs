//! Common result types for database operations
//!
//! These types standardize how we handle database responses,
//! reducing boilerplate in repositories.

use serde::Deserialize;
use surrealdb::RecordId;

/// Count result from aggregation queries
///
/// Used with `SELECT count() ... GROUP ALL` queries.
///
/// # Example
///
/// ```rust,ignore
/// let result: Option<CountResult> = db
///     .query("SELECT count() FROM tasks WHERE user = $user GROUP ALL")
///     .bind(("user", user_id))
///     .await?
///     .take(0)?;
///
/// let total = result.map(|r| r.count).unwrap_or(0);
/// ```
#[derive(Debug, Clone, Deserialize)]
pub struct CountResult {
    pub count: i64,
}

impl CountResult {
    /// Get count or 0 from an Option<CountResult>
    pub fn unwrap_or_zero(opt: Option<Self>) -> i64 {
        opt.map(|r| r.count).unwrap_or(0)
    }
}

/// ID-only result from INSERT/UPDATE queries
///
/// Used when queries return just the record ID.
///
/// # Example
///
/// ```rust,ignore
/// let result: Option<IdResult> = db
///     .query("INSERT INTO tasks {...} RETURN id")
///     .await?
///     .take(0)?;
///
/// let id = result.ok_or(AppError::Internal)?.id_string();
/// ```
#[derive(Debug, Clone, Deserialize)]
pub struct IdResult {
    pub id: RecordId,
}

impl IdResult {
    /// Get the ID as a string (table:id format)
    pub fn id_string(&self) -> String {
        self.id.to_string()
    }
}


