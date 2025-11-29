//! Health check feature
//!
//! Simple health check endpoints for monitoring and load balancers.

pub mod handler;

pub use handler::{health_check, routes};
