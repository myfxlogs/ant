-- Phase 2 debate v2 flow: relax check constraints so the v2 status/turn type
-- values added in service/debate_v2_service.go can be persisted.

-- 1. Session status: drop the restrictive IN-list and replace with a regex that
--    accepts the legacy values plus any 'v2:...' string (v2:intent,
--    v2:agent:<key>, v2:code, v2:done).
ALTER TABLE debate_sessions
    DROP CONSTRAINT IF EXISTS debate_sessions_status_check;

ALTER TABLE debate_sessions
    ADD CONSTRAINT debate_sessions_status_check
    CHECK (
        status IN (
            'idle',
            'clarifying',
            'intent_confirm',
            'debating',
            'consensus',
            'code_proposal',
            'saved',
            'archived'
        )
        OR status LIKE 'v2:%'
    );

-- 2. Turn type: allow the v2_* turn types alongside the legacy set.
ALTER TABLE debate_turns
    DROP CONSTRAINT IF EXISTS debate_turns_type_check;

ALTER TABLE debate_turns
    ADD CONSTRAINT debate_turns_type_check
    CHECK (type IN (
        'user_intent',
        'clarify_question',
        'clarify_answer',
        'intent_spec',
        'agent_opinion',
        'user_feedback',
        'consensus',
        'code_proposal',
        'system_note',
        'v2_user',
        'v2_assistant',
        'v2_code'
    ));
