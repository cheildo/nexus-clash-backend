-- This table stores game-specific player data.
CREATE TABLE IF NOT EXISTS profiles (
    -- 'user_id' is both the Primary Key and a Foreign Key referencing the 'users' table.
    -- This creates a strict one-to-one relationship, ensuring every profile is linked to exactly one user.
    -- 'ON DELETE CASCADE' means if a user is deleted from the 'users' table, their corresponding profile is automatically deleted.
    user_id UUID PRIMARY KEY,

    -- 'username' is stored here again for quick lookups and to be the canonical display name in the game.
    -- It must be unique.
    username VARCHAR(50) UNIQUE NOT NULL,

    -- 'level' represents the player's progression. It defaults to 1 for new profiles.
    -- 'CHECK (level > 0)' is a constraint to ensure the level is always a positive integer.
    level INT NOT NULL DEFAULT 1 CHECK (level > 0),

    -- 'stats_kills', 'stats_deaths', 'stats_assists', etc., store the player's lifetime gameplay statistics.
    -- They default to 0.
    stats_kills INT NOT NULL DEFAULT 0,
    stats_deaths INT NOT NULL DEFAULT 0,
    stats_assists INT NOT NULL DEFAULT 0,
    stats_wins INT NOT NULL DEFAULT 0,
    stats_losses INT NOT NULL DEFAULT 0,

    -- Timestamps for record-keeping, consistent with our 'users' table.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key constraint definition.
    CONSTRAINT fk_user
        FOREIGN KEY(user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
);

-- Re-using the same trigger function from the first migration to handle 'updated_at' automatically.
CREATE OR REPLACE TRIGGER set_profiles_updated_at
BEFORE UPDATE ON profiles
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();