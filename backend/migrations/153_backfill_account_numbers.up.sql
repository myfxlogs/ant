-- Migration 153: Backfill account numbers for users that don't have one.
-- Uses a PL/pgSQL block to iterate existing users and assign the next available
-- account number from the valid digit set.
-- Valid chars: first {1,2,3,5,6,8,9}, rest {0,1,2,3,5,6,8,9}. No 4 or 7.

DO $$
DECLARE
    valid_first  TEXT[] := ARRAY['1','2','3','5','6','8','9'];
    valid_rest   TEXT[] := ARRAY['0','1','2','3','5','6','8','9'];
    rec          RECORD;
    candidate    TEXT;
    attempts     INT;
    exhausted    BOOLEAN := false;
BEGIN
    FOR rec IN
        SELECT id FROM users WHERE account_number IS NULL AND deleted_at IS NULL
        ORDER BY created_at ASC
    LOOP
        exhausted := true;
        <<outer>>
        FOR f IN 1..array_length(valid_first, 1) LOOP
            FOR a IN 1..array_length(valid_rest, 1) LOOP
                FOR b IN 1..array_length(valid_rest, 1) LOOP
                    FOR c IN 1..array_length(valid_rest, 1) LOOP
                        FOR d IN 1..array_length(valid_rest, 1) LOOP
                            candidate := valid_first[f] || valid_rest[a] || valid_rest[b] || valid_rest[c] || valid_rest[d];
                            -- Check if this candidate is taken
                            IF NOT EXISTS (SELECT 1 FROM users WHERE account_number = candidate) THEN
                                UPDATE users SET account_number = candidate WHERE id = rec.id;
                                exhausted := false;
                                EXIT outer;
                            END IF;
                        END LOOP;
                    END LOOP;
                END LOOP;
            END LOOP;
        END LOOP;
        IF exhausted THEN
            RAISE WARNING 'Account number space exhausted for user %', rec.id;
        END IF;
    END LOOP;
END
$$;
