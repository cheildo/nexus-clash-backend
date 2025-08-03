-- We create a table to store user authentication information.
CREATE TABLE IF NOT EXISTS users (
    -- 'id' is the primary key, a UUID automatically generated for each new user.
    -- This ensures a unique, non-sequential identifier for each user.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 'email' must be unique and is used for login. It is stored in lowercase to prevent duplicates.
    email VARCHAR(255) UNIQUE NOT NULL,

    -- 'username' must also be unique.
    username VARCHAR(50) UNIQUE NOT NULL,

    -- 'password_hash' stores the securely hashed password, not the plaintext password.
    password_hash VARCHAR(255) NOT NULL,

    -- 'created_at' and 'updated_at' are timestamps for record-keeping.
    -- 'created_at' defaults to the current time on insertion.
    -- 'updated_at' is managed by a trigger to update whenever the row is changed.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- This trigger automatically updates the 'updated_at' timestamp on any row update.
CREATE OR REPLACE FUNCTION set_updated_at_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER set_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();