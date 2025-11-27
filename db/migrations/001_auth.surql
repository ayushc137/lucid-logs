-- Migration: 001_auth
-- Description: Authentication scope and users table
-- Created: 2025-11-27

-- Authentication scope for end-users
-- JWT tokens are valid for 1 hour, sessions for 1 day
DEFINE ACCESS account ON DATABASE TYPE RECORD 
  SIGNUP ( 
    CREATE user SET 
      email = $user, 
      pass = crypto::argon2::generate($pass) 
  ) 
  SIGNIN ( 
    SELECT * FROM user 
    WHERE email = $user 
    AND crypto::argon2::compare(pass, $pass) 
  ) 
  WITH JWT ALGORITHM HS512 KEY "${JWT_SECRET}" 
  DURATION FOR TOKEN 1h, FOR SESSION 1d;

-- Users table
DEFINE TABLE user SCHEMAFULL;
DEFINE FIELD email ON user TYPE string ASSERT string::is::email($value);
DEFINE FIELD pass ON user TYPE string;

-- Unique email constraint
DEFINE INDEX idx_user_email ON TABLE user COLUMNS email UNIQUE;

