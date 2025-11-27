/// Helper to ensure SurrealDB record IDs have the table prefix
pub fn ensure_record_id(id: &str, table: &str) -> String {
    if id.contains(':') {
        id.to_string()
    } else {
        format!("{}:{}", table, id)
    }
}

/// Count result helper struct for aggregation queries
#[derive(Debug, serde::Deserialize)]
pub struct CountResult {
    pub count: i64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ensure_record_id_with_prefix() {
        assert_eq!(ensure_record_id("tasks:123", "tasks"), "tasks:123");
    }

    #[test]
    fn test_ensure_record_id_without_prefix() {
        assert_eq!(ensure_record_id("123", "tasks"), "tasks:123");
    }
}
