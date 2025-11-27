-- SurrealDB schema for Daily Journal App

-- Authentication scope for end-users
DEFINE ACCESS account ON DATABASE TYPE RECORD SIGNUP ( CREATE user SET email = $user, pass = crypto::argon2::generate($pass) ) SIGNIN ( SELECT * FROM user WHERE email = $user AND crypto::argon2::compare(pass, $pass) ) WITH JWT ALGORITHM HS512 KEY "${JWT_SECRET}" DURATION FOR TOKEN 1h, FOR SESSION 1d;

-- Users table
DEFINE TABLE user SCHEMAFULL;
DEFINE FIELD email ON user TYPE string ASSERT string::is::email($value);
DEFINE FIELD pass ON user TYPE string;
DEFINE INDEX idx_user_email ON TABLE user COLUMNS email UNIQUE;

-- Lookup tables (temporarily disabled until implemented)
-- DEFINE TABLE mood SCHEMAFULL;
-- DEFINE FIELD name ON mood TYPE string ASSERT string::len($value) > 0;
-- DEFINE FIELD emoji ON mood TYPE option<string>;
-- DEFINE FIELD description ON mood TYPE option<string>;

-- DEFINE TABLE category SCHEMAFULL;
-- DEFINE FIELD name ON category TYPE string ASSERT string::len($value) > 0;
-- DEFINE FIELD color ON category TYPE option<string>;

-- DEFINE TABLE template SCHEMAFULL;
-- DEFINE FIELD name ON template TYPE string ASSERT string::len($value) > 0;
-- DEFINE FIELD icon ON template TYPE option<string>;
-- DEFINE FIELD color ON template TYPE option<string>;
-- DEFINE FIELD default_duration ON template TYPE option<duration>;

-- Tasks table (core entity)
-- Permissions: $auth is NONE for root sessions, otherwise check ownership
DEFINE TABLE tasks SCHEMAFULL
  PERMISSIONS
    FOR select WHERE $auth = NONE OR created_by = $auth.id
    FOR create WHERE $auth = NONE OR created_by = $auth.id
    FOR update WHERE $auth = NONE OR created_by = $auth.id
    FOR delete WHERE $auth = NONE OR created_by = $auth.id;

DEFINE FIELD title ON tasks TYPE string ASSERT string::len(string::trim($value)) > 0;
DEFINE FIELD journal ON tasks TYPE option<string>;
DEFINE FIELD start_date ON tasks TYPE datetime ASSERT $value != NONE;
DEFINE FIELD end_date ON tasks TYPE datetime ASSERT $value >= start_date;
DEFINE FIELD is_completed ON tasks TYPE bool DEFAULT false;
DEFINE FIELD priority ON tasks TYPE number DEFAULT 0;
DEFINE FIELD source ON tasks TYPE string VALUE $value ?? "manual" ASSERT $value = "manual";
DEFINE FIELD note ON tasks TYPE option<string>;
DEFINE FIELD positives ON tasks TYPE array<string> DEFAULT [];
DEFINE FIELD negatives ON tasks TYPE array<string> DEFAULT [];
DEFINE FIELD planned ON tasks TYPE bool VALUE $value ?? ((start_date >= time::now()) && start_date != end_date);
-- DEFINE FIELD mood_id ON tasks TYPE option<record<mood>>;
-- DEFINE FIELD category_id ON tasks TYPE option<record<category>>;
-- DEFINE FIELD template_id ON tasks TYPE option<record<template>>;
DEFINE FIELD created_at ON tasks TYPE option<datetime> VALUE $value OR time::now();
DEFINE FIELD updated_at ON tasks TYPE option<datetime> VALUE time::now();
DEFINE FIELD deleted_at ON tasks TYPE option<datetime> VALUE $value ?? NONE;
DEFINE FIELD created_by ON tasks TYPE string ASSERT string::len($value) > 0;
DEFINE FIELD updated_by ON tasks TYPE string ASSERT string::len($value) > 0;

DEFINE INDEX idx_tasks_user ON TABLE tasks COLUMNS created_by;
DEFINE INDEX idx_tasks_time ON TABLE tasks COLUMNS start_date;
DEFINE INDEX idx_tasks_status ON TABLE tasks COLUMNS is_completed;

-- Soft delete helper (optionally filter deleted_at IS NONE in queries)

