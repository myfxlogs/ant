-- Revert v2 check constraint relaxations.
ALTER TABLE debate_sessions
    DROP CONSTRAINT IF EXISTS debate_sessions_status_check;

ALTER TABLE debate_sessions
    ADD CONSTRAINT debate_sessions_status_check
    CHECK (status IN (
        'idle','clarifying','intent_confirm','debating',
        'consensus','code_proposal','saved','archived'
    ));

ALTER TABLE debate_turns
    DROP CONSTRAINT IF EXISTS debate_turns_type_check;

ALTER TABLE debate_turns
    ADD CONSTRAINT debate_turns_type_check
    CHECK (type IN (
        'user_intent','clarify_question','clarify_answer','intent_spec',
        'agent_opinion','user_feedback','consensus','code_proposal','system_note'
    ));
