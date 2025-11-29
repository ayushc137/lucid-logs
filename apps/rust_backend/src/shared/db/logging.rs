use surrealdb::Error;

/// Extension helpers for logging SurrealDB errors at the call site.
pub trait DbResultExt<T> {
    /// Logs the error using the provided closure but keeps the original result intact.
    fn log_db_err(self, log: impl FnOnce(&Error)) -> Result<T, Error>;
}

impl<T> DbResultExt<T> for Result<T, Error> {
    fn log_db_err(self, log: impl FnOnce(&Error)) -> Result<T, Error> {
        if let Err(ref err) = self {
            log(err);
        }
        self
    }
}

