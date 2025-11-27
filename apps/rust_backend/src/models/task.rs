use chrono::{DateTime, NaiveDate, Utc};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use surrealdb::RecordId;
use utoipa::ToSchema;
use validator::Validate;

use super::category::Category;

pub const SOURCE_MANUAL: &str = "manual";

/// Custom deserializer that converts null to empty string
fn null_to_empty_string<'de, D>(deserializer: D) -> Result<String, D::Error>
where
    D: Deserializer<'de>,
{
    let opt: Option<String> = Option::deserialize(deserializer)?;
    Ok(opt.unwrap_or_default())
}

/// Custom deserializer that converts null to empty vec
fn null_to_empty_vec<'de, D>(deserializer: D) -> Result<Vec<String>, D::Error>
where
    D: Deserializer<'de>,
{
    let opt: Option<Vec<String>> = Option::deserialize(deserializer)?;
    Ok(opt.unwrap_or_default())
}

/// Custom deserializer for SurrealDB RecordId -> Option<String>
fn deserialize_record_id<'de, D>(deserializer: D) -> Result<Option<String>, D::Error>
where
    D: Deserializer<'de>,
{
    Option::<RecordId>::deserialize(deserializer).map(|opt| opt.map(|id| id.to_string()))
}

/// Custom serializer that just passes through the Option<String>
fn serialize_record_id<S>(value: &Option<String>, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    match value {
        Some(s) => serializer.serialize_some(s),
        None => serializer.serialize_none(),
    }
}

/// Task model representing a daily task or event
#[derive(Debug, Serialize, Deserialize, Clone, ToSchema)]
#[schema(example = json!({
    "id": "tasks:abc123",
    "title": "Plan tomorrow",
    "journal": "Capture high-level goals",
    "start_date": "2025-11-24T09:00:00Z",
    "end_date": "2025-11-25T17:00:00Z",
    "completed": false,
    "priority": 1,
    "source": "manual",
    "note": "Focus on top priorities",
    "positives": ["Felt great", "In flow"],
    "negatives": ["Got distracted"],
    "category": {
        "id": "categories:work123",
        "name": "Work",
        "color": "#3B82F6"
    },
    "created_at": "2025-11-24T10:00:00Z",
    "updated_at": "2025-11-24T10:00:00Z"
}))]
pub struct Task {
    /// Task ID (SurrealDB record ID like "tasks:abc123")
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        deserialize_with = "deserialize_record_id",
        serialize_with = "serialize_record_id"
    )]
    #[schema(example = "tasks:abc123")]
    pub id: Option<String>,

    /// Task title
    #[schema(example = "Plan tomorrow")]
    pub title: String,

    /// Journal entry / detailed notes
    #[serde(default, deserialize_with = "null_to_empty_string")]
    #[schema(example = "Capture high-level goals")]
    pub journal: String,

    /// Start date and time (ISO8601 format)
    #[schema(value_type = String, format = DateTime, example = "2025-11-24T09:00:00Z")]
    pub start_date: DateTime<Utc>,

    /// End date and time (ISO8601 format)
    #[schema(value_type = String, format = DateTime, example = "2025-11-25T17:00:00Z")]
    pub end_date: DateTime<Utc>,

    /// Whether the task is completed
    #[serde(default)]
    #[schema(example = false)]
    pub completed: bool,

    /// Priority level (negative = wishes, higher = higher priority)
    #[serde(default)]
    #[schema(example = 1)]
    pub priority: i32,

    /// Source of the task (default: "manual")
    #[serde(default = "default_source")]
    #[schema(example = "manual")]
    pub source: String,

    /// Additional notes
    #[serde(default, deserialize_with = "null_to_empty_string")]
    #[schema(example = "Focus on top priorities")]
    pub note: String,

    /// Positive comments about the task
    #[serde(default, deserialize_with = "null_to_empty_vec")]
    #[schema(example = json!(["Felt great", "In flow"]))]
    pub positives: Vec<String>,

    /// Negative comments or issues
    #[serde(default, deserialize_with = "null_to_empty_vec")]
    #[schema(example = json!(["Got distracted"]))]
    pub negatives: Vec<String>,

    /// Category (populated via FETCH category)
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub category: Option<Category>,

    // === System-managed fields (read-only, set by DB) ===
    /// Creation timestamp (system-managed, read-only)
    #[serde(default = "Utc::now")]
    #[schema(value_type = String, format = DateTime, read_only = true)]
    pub created_at: DateTime<Utc>,

    /// Last update timestamp (system-managed, read-only)
    #[serde(default = "Utc::now")]
    #[schema(value_type = String, format = DateTime, read_only = true)]
    pub updated_at: DateTime<Utc>,

    /// Soft delete timestamp (system-managed, read-only)
    #[serde(default, skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<String>, format = DateTime, read_only = true)]
    pub deleted_at: Option<DateTime<Utc>>,

    /// User ID who created the task (system-managed, read-only, hidden from API)
    #[serde(default, skip_serializing, rename = "created_by")]
    #[schema(example = "user:xyz789", read_only = true)]
    pub _created_by: String,

    /// User ID who last updated the task (system-managed, read-only, hidden from API)
    #[serde(default, skip_serializing, rename = "updated_by")]
    #[schema(example = "user:xyz789", read_only = true)]
    pub _updated_by: String,
}

fn default_source() -> String {
    SOURCE_MANUAL.to_string()
}

/// Custom deserializer for datetime that accepts multiple formats
#[derive(Debug, Clone, ToSchema)]
#[schema(value_type = String, format = DateTime, example = "2025-11-24T09:00:00Z")]
pub struct DateTimeInput(pub DateTime<Utc>);

impl<'de> Deserialize<'de> for DateTimeInput {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let s: String = String::deserialize(deserializer)?;

        // Reject empty or null strings - silent coercion to epoch causes data corruption
        if s.is_empty() {
            return Err(serde::de::Error::custom("datetime is required"));
        }
        if s == "null" {
            return Err(serde::de::Error::custom("datetime cannot be null"));
        }

        // Try YYYY-MM-DD format (date only, assumes midnight UTC)
        if let Ok(date) = NaiveDate::parse_from_str(&s, "%Y-%m-%d") {
            let datetime = date
                .and_hms_opt(0, 0, 0)
                .map(|naive| DateTime::from_naive_utc_and_offset(naive, Utc))
                .ok_or_else(|| serde::de::Error::custom("Invalid date"))?;
            return Ok(DateTimeInput(datetime));
        }

        // Try RFC3339/ISO8601 with time
        if let Ok(dt) = DateTime::parse_from_rfc3339(&s) {
            return Ok(DateTimeInput(dt.with_timezone(&Utc)));
        }

        // Try common datetime format without timezone
        if let Ok(naive) = chrono::NaiveDateTime::parse_from_str(&s, "%Y-%m-%dT%H:%M:%S") {
            return Ok(DateTimeInput(DateTime::from_naive_utc_and_offset(
                naive, Utc,
            )));
        }

        Err(serde::de::Error::custom(format!(
            "Invalid datetime format '{}'. Expected ISO8601 format like '2025-11-24T09:00:00Z' or date '2025-11-24'",
            s
        )))
    }
}

impl Serialize for DateTimeInput {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.0.to_rfc3339())
    }
}

impl DateTimeInput {
    pub fn time_value(&self) -> DateTime<Utc> {
        self.0
    }
}

/// Request payload for creating a new task
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "title": "Plan tomorrow",
    "journal": "Capture high-level goals",
    "start_date": "2025-11-24T09:00:00Z",
    "end_date": "2025-11-25T17:00:00Z",
    "priority": 1,
    "source": "manual",
    "note": "Focus on top priorities",
    "positives": ["Great progress"],
    "negatives": ["Some distractions"],
    "category_id": "categories:work123"
}))]
pub struct CreateTaskRequest {
    /// Task title (required)
    #[validate(length(min = 1, message = "Title is required"))]
    #[schema(example = "Plan tomorrow")]
    pub title: String,

    /// Journal entry / detailed notes
    #[serde(default)]
    #[schema(example = "Capture high-level goals")]
    pub journal: String,

    /// Start date/time (required, format: ISO8601 e.g., 2025-11-24T09:00:00Z or just 2025-11-24)
    #[schema(value_type = String, format = DateTime, example = "2025-11-24T09:00:00Z")]
    pub start_date: DateTimeInput,

    /// End date/time (required, must be >= start_date)
    #[schema(value_type = String, format = DateTime, example = "2025-11-25T17:00:00Z")]
    pub end_date: DateTimeInput,

    /// Priority level (negative = wishes, higher = higher priority)
    #[serde(default)]
    #[validate(range(min = -100, max = 100, message = "Priority must be between -100 and 100"))]
    #[schema(example = 1)]
    pub priority: i32,

    /// Source of the task (only "manual" is allowed)
    #[serde(default)]
    #[schema(example = "manual")]
    pub source: Option<String>,

    /// Additional notes
    #[serde(default)]
    #[schema(example = "Focus on top priorities")]
    pub note: Option<String>,

    /// Positive comments
    #[serde(default)]
    #[schema(example = json!(["Great progress"]))]
    pub positives: Option<Vec<String>>,

    /// Negative comments
    #[serde(default)]
    #[schema(example = json!(["Some distractions"]))]
    pub negatives: Option<Vec<String>>,

    /// Category ID to link (optional)
    #[serde(default)]
    #[schema(example = "categories:work123")]
    pub category_id: Option<String>,
}

/// Request payload for updating an existing task
#[derive(Debug, Deserialize, Validate, ToSchema)]
#[schema(example = json!({
    "title": "Updated task title",
    "completed": true,
    "priority": 2,
    "category_id": "categories:personal456"
}))]
pub struct UpdateTaskRequest {
    /// New title (optional, but if provided must be non-empty)
    #[validate(custom(function = "validate_optional_title"))]
    #[schema(example = "Updated task title")]
    pub title: Option<String>,

    /// New journal entry (optional)
    #[schema(example = "Updated journal")]
    pub journal: Option<String>,

    /// New start date/time (format: ISO8601)
    #[schema(value_type = Option<String>, format = DateTime, example = "2025-11-26T09:00:00Z")]
    pub start_date: Option<DateTimeInput>,

    /// New end date/time (must be >= start_date)
    #[schema(value_type = Option<String>, format = DateTime, example = "2025-11-27T17:00:00Z")]
    pub end_date: Option<DateTimeInput>,

    /// Mark as completed
    #[schema(example = true)]
    pub completed: Option<bool>,

    /// New priority level (-100 to 100)
    #[validate(custom(function = "validate_optional_priority"))]
    #[schema(example = 2)]
    pub priority: Option<i32>,

    /// Updated notes
    #[schema(example = "Updated notes")]
    pub note: Option<String>,

    /// Updated positive comments
    pub positives: Option<Vec<String>>,

    /// Updated negative comments
    pub negatives: Option<Vec<String>>,

    /// Category ID to link (use null or empty string to remove)
    #[schema(example = "categories:personal456")]
    pub category_id: Option<String>,
}

/// Validate that optional title is non-empty if provided
fn validate_optional_title(title: &str) -> Result<(), validator::ValidationError> {
    if title.trim().is_empty() {
        return Err(validator::ValidationError::new("title_empty")
            .with_message("Title cannot be empty".into()));
    }
    Ok(())
}

/// Validate that optional priority is within range if provided
fn validate_optional_priority(priority: i32) -> Result<(), validator::ValidationError> {
    if !(-100..=100).contains(&priority) {
        return Err(validator::ValidationError::new("priority_range")
            .with_message("Priority must be between -100 and 100".into()));
    }
    Ok(())
}
