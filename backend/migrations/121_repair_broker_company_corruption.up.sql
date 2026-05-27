-- 121: Repair broker_company corruption caused by UpdateAccount SQL parameter shift.
-- The bug mapped user_id (UUID) into broker_company. Detect affected rows and
-- flag them for re-binding since the original broker_company cannot be recovered.

-- Detect rows where broker_company looks like a UUID (corruption indicator)
-- Mark them needs_rebind so users re-bind and restore correct broker_company.
UPDATE mt_accounts
   SET account_status = 'needs_rebind',
       updated_at = CURRENT_TIMESTAMP
 WHERE broker_company ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
