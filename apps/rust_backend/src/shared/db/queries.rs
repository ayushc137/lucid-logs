//! Centralized query registry for SurrealDB
//!
//! All SQL queries are defined here for:
//! - **Maintainability**: Single source of truth for query logic
//! - **Consistency**: Reuse common patterns across repositories
//! - **Optimization**: Easy to review and tune queries
//! - **Documentation**: Clear overview of database access patterns
//!
//! # Naming Convention
//!
//! - `CREATE_*`: Insert new records
//! - `SELECT_*`: Read operations
//! - `UPDATE_*`: Modify existing records
//! - `DELETE_*`: Remove records (usually soft delete)
//! - `COUNT_*`: Aggregation queries
//! - `LIST_*`: Paginated list queries
//!
//! # Usage
//!
//! ```rust,ignore
//! use crate::shared::db::queries::tasks;
//!
//! let result = db
//!     .query(tasks::LIST_BY_USER)
//!     .bind(("user", user_id))
//!     .bind(("limit", 25))
//!     .bind(("offset", 0))
//!     .await?;
//! ```

// =============================================================================
// TASK QUERIES
// =============================================================================

pub mod tasks {
    //! Task-related database queries
    //!
    //! All task queries enforce ownership via `created_by` and exclude
    //! soft-deleted records via `deleted_at = NONE`.

    /// Select a task by ID with category populated
    ///
    /// # Bindings
    /// - `$id`: Task record ID (thing)
    /// - `$user`: User ID for ownership check (string)
    pub const SELECT_BY_ID: &str = r"
        SELECT * FROM type::thing($id)
        WHERE created_by = $user AND deleted_at = NONE
        FETCH category
    ";

    /// Count tasks for a user (excludes soft-deleted)
    ///
    /// # Bindings
    /// - `$user`: User ID (string)
    pub const COUNT_BY_USER: &str = r"
        SELECT count() FROM tasks
        WHERE created_by = $user AND deleted_at = NONE
        GROUP ALL
    ";

    /// List tasks for a user with pagination and category populated
    ///
    /// # Bindings
    /// - `$user`: User ID (string)
    /// - `$limit`: Max results (int)
    /// - `$offset`: Skip count (int)
    pub const LIST_BY_USER: &str = r"
        SELECT * FROM tasks
        WHERE created_by = $user AND deleted_at = NONE
        ORDER BY start_date DESC
        LIMIT $limit START $offset
        FETCH category
    ";

    /// Soft-delete a task
    ///
    /// # Bindings
    /// - `$id`: Task record ID
    /// - `$user`: User ID
    /// - `$now`: Current timestamp (ISO8601 string)
    pub const SOFT_DELETE: &str = r"
        UPDATE type::thing($id) SET
            deleted_at = type::datetime($now),
            updated_by = $user,
            updated_at = time::now()
        WHERE created_by = $user AND deleted_at = NONE
        RETURN id
    ";
}

// =============================================================================
// CATEGORY QUERIES
// =============================================================================

pub mod categories {
    //! Category-related database queries

    /// Insert a new category
    ///
    /// # Bindings
    /// - `$name`: Category name (string)
    /// - `$color`: Color value (string)
    /// - `$user`: User ID (string)
    pub const CREATE: &str = r"
        INSERT INTO categories (name, color, created_by, updated_by)
        VALUES ($name, $color, $user, $user)
    ";

    /// Select a category by ID
    ///
    /// # Bindings
    /// - `$id`: Category record ID (thing)
    /// - `$user`: User ID
    pub const SELECT_BY_ID: &str = r"
        SELECT id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
        FROM type::thing($id)
        WHERE created_by = $user AND deleted_at = NONE
    ";

    /// Select a category by name for a user
    ///
    /// # Bindings
    /// - `$name`: Category name
    /// - `$user`: User ID
    pub const SELECT_BY_NAME: &str = r"
        SELECT id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
        FROM categories
        WHERE name = $name AND created_by = $user AND deleted_at = NONE
    ";

    /// Count categories for a user
    ///
    /// # Bindings
    /// - `$user`: User ID
    pub const COUNT_BY_USER: &str = r"
        SELECT count() FROM categories
        WHERE created_by = $user AND deleted_at = NONE
        GROUP ALL
    ";

    /// List categories for a user with pagination
    ///
    /// # Bindings
    /// - `$user`: User ID
    /// - `$limit`: Max results
    /// - `$offset`: Skip count
    pub const LIST_BY_USER: &str = r"
        SELECT id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
        FROM categories
        WHERE created_by = $user AND deleted_at = NONE
        ORDER BY name ASC
        LIMIT $limit START $offset
    ";

    /// Update category fields
    ///
    /// # Bindings
    /// - `$id`: Category record ID
    /// - `$name`: New name
    /// - `$color`: New color
    /// - `$user`: User ID
    pub const UPDATE: &str = r"
        UPDATE type::thing($id) SET
            name = $name,
            color = $color,
            updated_by = $user,
            updated_at = time::now()
        WHERE created_by = $user AND deleted_at = NONE
        RETURN id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
    ";

    /// Soft-delete a category
    ///
    /// # Bindings
    /// - `$id`: Category record ID
    /// - `$user`: User ID
    /// - `$now`: Current timestamp
    pub const SOFT_DELETE: &str = r"
        UPDATE type::thing($id) SET
            deleted_at = type::datetime($now),
            updated_by = $user,
            updated_at = time::now()
        WHERE created_by = $user AND deleted_at = NONE
        RETURN id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
    ";
}


