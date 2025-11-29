//! Type-safe record ID wrappers for SurrealDB
//!
//! These types provide compile-time safety for table-specific record IDs,
//! preventing accidental mixing of IDs from different tables.
//!
//! # Design Philosophy
//!
//! Instead of passing raw strings everywhere, we use newtype wrappers that:
//! 1. Encode the table name at the type level
//! 2. Handle the `table:id` format automatically
//! 3. Implement common traits for ergonomic usage
//!
//! # Examples
//!
//! ```rust,ignore
//! // Create type-safe IDs
//! let task_id = TaskId::new("abc123");
//! let category_id = CategoryId::new("work");
//!
//! // IDs automatically format correctly
//! assert_eq!(task_id.to_string(), "tasks:abc123");
//!
//! // Parse from full record ID strings
//! let parsed = TaskId::parse("tasks:xyz789")?;
//!
//! // Use in queries
//! db.query("SELECT * FROM type::thing($id)")
//!     .bind(("id", task_id.as_thing()))
//!     .await?;
//! ```

use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::fmt;
use surrealdb::sql::Thing;

// =============================================================================
// TABLE NAME CONSTANTS
// =============================================================================


// =============================================================================
// GENERIC RECORD ID
// =============================================================================

/// Generic record ID that can represent any table's record.
///
/// Use the specific types (`TaskId`, `CategoryId`) when the table is known.
/// Use this type for dynamic scenarios or generic code.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct RecordId {
    table: String,
    id: String,
}

impl RecordId {
    /// Create a new record ID from table and ID components
    pub fn new(table: impl Into<String>, id: impl Into<String>) -> Self {
        Self {
            table: table.into(),
            id: id.into(),
        }
    }

    /// Parse a record ID from a string like "table:id"
    pub fn parse(s: &str) -> Option<Self> {
        let (table, id) = s.split_once(':')?;
        Some(Self::new(table, id))
    }

    /// Get the table name
    pub fn table(&self) -> &str {
        &self.table
    }

    /// Get the ID portion (without table prefix)
    pub fn id(&self) -> &str {
        &self.id
    }

    /// Convert to a SurrealDB Thing for use in queries
    pub fn as_thing(&self) -> Thing {
        Thing::from((self.table.as_str(), self.id.as_str()))
    }

    /// Get the full record ID string (table:id)
    pub fn full_id(&self) -> String {
        format!("{}:{}", self.table, self.id)
    }
}

impl fmt::Display for RecordId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}:{}", self.table, self.id)
    }
}

impl From<Thing> for RecordId {
    fn from(thing: Thing) -> Self {
        Self::new(thing.tb.to_string(), thing.id.to_string())
    }
}

impl From<surrealdb::RecordId> for RecordId {
    fn from(id: surrealdb::RecordId) -> Self {
        let s = id.to_string();
        Self::parse(&s).unwrap_or_else(|| Self::new("unknown", s))
    }
}

// =============================================================================
// TABLE-SPECIFIC RECORD ID MACRO
// =============================================================================

/// Macro to generate table-specific record ID types
macro_rules! define_record_id {
    (
        $(#[$meta:meta])*
        $name:ident => $table:expr
    ) => {
        $(#[$meta])*
        #[derive(Debug, Clone, PartialEq, Eq, Hash)]
        pub struct $name(String);

        impl $name {
            /// Table name for this record type
            pub const TABLE: &'static str = $table;

            /// Create a new record ID from just the ID portion
            ///
            /// If the input already contains a colon, it's parsed as a full ID.
            /// Otherwise, it's treated as just the ID portion.
            pub fn new(id: impl Into<String>) -> Self {
                let id_str = id.into();
                // Strip table prefix if present
                let clean_id = id_str
                    .strip_prefix(concat!($table, ":"))
                    .map(String::from)
                    .unwrap_or(id_str);
                Self(clean_id)
            }

            /// Parse from a full record ID string (table:id)
            ///
            /// Returns `None` if the table doesn't match
            pub fn parse(s: &str) -> Option<Self> {
                let (table, id) = s.split_once(':')?;
                if table == $table {
                    Some(Self(id.to_string()))
                } else {
                    None
                }
            }

            /// Get just the ID portion (without table prefix)
            pub fn id(&self) -> &str {
                &self.0
            }

            /// Get the full record ID string (table:id)
            pub fn full_id(&self) -> String {
                format!("{}:{}", $table, self.0)
            }

            /// Convert to a SurrealDB Thing for use in raw queries
            pub fn as_thing(&self) -> Thing {
                Thing::from(($table, self.0.as_str()))
            }

            /// Convert to a tuple for use with fluent builders (select, update, delete)
            ///
            /// SurrealDB 2.x fluent API accepts `(table, id)` tuples
            pub fn as_key(&self) -> (&'static str, &str) {
                ($table, &self.0)
            }

            /// Convert to generic RecordId
            pub fn as_record_id(&self) -> RecordId {
                RecordId::new($table, &self.0)
            }
        }

        impl fmt::Display for $name {
            fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                write!(f, "{}:{}", $table, self.0)
            }
        }

        impl From<$name> for Thing {
            fn from(id: $name) -> Self {
                id.as_thing()
            }
        }

        impl From<$name> for RecordId {
            fn from(id: $name) -> Self {
                id.as_record_id()
            }
        }

        impl From<$name> for String {
            fn from(id: $name) -> Self {
                id.full_id()
            }
        }

        impl AsRef<str> for $name {
            fn as_ref(&self) -> &str {
                &self.0
            }
        }

        // Serde: serialize as full "table:id" string
        impl Serialize for $name {
            fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
            where
                S: Serializer,
            {
                serializer.serialize_str(&self.full_id())
            }
        }

        // Serde: deserialize from "table:id" or just "id"
        impl<'de> Deserialize<'de> for $name {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where
                D: Deserializer<'de>,
            {
                let s = String::deserialize(deserializer)?;
                Ok(Self::new(s))
            }
        }
    };
}

// =============================================================================
// TABLE-SPECIFIC RECORD ID TYPES
// =============================================================================

define_record_id! {
    /// Type-safe task record ID
    ///
    /// # Examples
    /// ```rust,ignore
    /// let id = TaskId::new("abc123");
    /// assert_eq!(id.full_id(), "tasks:abc123");
    /// ```
    TaskId => "tasks"
}

define_record_id! {
    /// Type-safe category record ID
    ///
    /// # Examples
    /// ```rust,ignore
    /// let id = CategoryId::new("work");
    /// assert_eq!(id.full_id(), "categories:work");
    /// ```
    CategoryId => "categories"
}

define_record_id! {
    /// Type-safe user record ID
    ///
    /// # Examples
    /// ```rust,ignore
    /// let id = UserId::new("user123");
    /// assert_eq!(id.full_id(), "user:user123");
    /// ```
    UserId => "user"
}
