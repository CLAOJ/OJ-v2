-- cleanup_v2_tables.sql
--
-- Removes tables/columns created by the deleted OJ-v2 migrations 001-007
-- (roles/permissions system, audit log, refresh/password-reset/email-verify
-- tokens, Contest.max_submissions, Problem.suggestion_* review workflow).
-- Run MANUALLY, once, after backing up. Safe to skip: Django ignores these,
-- and current OJ-v2 code no longer references any of them.
--
-- Does NOT touch any RETAINED-V2 object (notification, notification_preference,
-- totp_device, backup_code, oauth_user_link, moss_result,
-- judge_commentrevision, judge_contestclarification, or the judge_solution
-- editorial columns) -- those are live features, kept on purpose. See
-- docs/schema-audit.md and scripts/v2_runtime_tables.sql.

DROP TABLE IF EXISTS judge_profile_roles;
DROP TABLE IF EXISTS judge_role_permissions;
DROP TABLE IF EXISTS judge_permission;
DROP TABLE IF EXISTS judge_role;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS refresh_token;
DROP TABLE IF EXISTS password_reset_token;
DROP TABLE IF EXISTS email_verification_token;
ALTER TABLE judge_contest DROP COLUMN max_submissions;
ALTER TABLE judge_problem
  DROP FOREIGN KEY fk_problem_suggestion_reviewed_by;
ALTER TABLE judge_problem
  DROP COLUMN suggestion_status,
  DROP COLUMN suggestion_notes,
  DROP COLUMN suggestion_reviewed_at,
  DROP COLUMN suggestion_reviewed_by_id;
