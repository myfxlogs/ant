-- Fix FK: user_factor_overrides.user_id → users(id) ON DELETE CASCADE
-- The original inline REFERENCES lacked an explicit ON DELETE clause (defaults to NO ACTION).
-- This prevents make check-fk from passing.

-- Drop existing unnamed FK constraints (inline REFERENCES creates auto-named constraints)
ALTER TABLE user_factor_overrides
  DROP CONSTRAINT IF EXISTS user_factor_overrides_user_id_fkey;

-- Re-add with CASCADE — deleting a user cascades to their factor overrides
ALTER TABLE user_factor_overrides
  ADD CONSTRAINT user_factor_overrides_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
