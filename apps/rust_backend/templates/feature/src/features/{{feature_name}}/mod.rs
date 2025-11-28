//! {{feature_name_pascal}} feature module
//!
//! Handles CRUD operations for {{feature_name}}s.

pub mod handler;
pub mod model;
pub mod repository;
pub mod service;

pub use handler::{
    create_{{feature_name}}, delete_{{feature_name}}, get_{{feature_name}}, list_{{feature_name}}s,
    protected_routes, routes, update_{{feature_name}},
};
pub use model::{Create{{feature_name_pascal}}Request, {{feature_name_pascal}}, Update{{feature_name_pascal}}Request};
pub use repository::{{feature_name_pascal}}Repository;
pub use service::{{{feature_name_pascal}}Service, {{feature_name_pascal}}ServiceImpl};

