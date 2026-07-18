-- v2_runtime_tables.sql
--
-- MANUAL, run once for a fresh side-by-side deployment (or any environment
-- that does not already have these objects). Provisions every additive
-- v2-only object OJ-v2 needs, on top of a database whose schema was created
-- by Django migrations (`cd OJ && python manage.py migrate`).
--
-- Django never creates, reads, or writes any of these tables/columns; they
-- are entirely owned and used by OJ-v2. All statements are idempotent
-- (CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS) so this script is
-- safe to re-run. Requires MySQL 8.0.29+ or MariaDB 10.3+ for the
-- `ADD COLUMN IF NOT EXISTS` clause used below.
--
-- See docs/schema-audit.md section 3 ("RETAINED-V2") for the full rationale
-- and decision record (2026-07-19) behind every object created here.

-- ---------------------------------------------------------------------
-- Notifications (models/notification.go)
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  type VARCHAR(20) NOT NULL,
  title VARCHAR(200) NOT NULL,
  message LONGTEXT NOT NULL,
  link VARCHAR(500) NULL,
  `read` TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_notification_user_id (user_id),
  KEY idx_notification_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notification_preference (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  email_on_submission_result TINYINT(1) NOT NULL DEFAULT 1,
  email_on_contest_start TINYINT(1) NOT NULL DEFAULT 1,
  email_on_ticket_reply TINYINT(1) NOT NULL DEFAULT 1,
  web_on_submission_result TINYINT(1) NOT NULL DEFAULT 1,
  web_on_contest_start TINYINT(1) NOT NULL DEFAULT 1,
  web_on_ticket_reply TINYINT(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (id),
  UNIQUE KEY uq_notification_preference_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- 2FA (models/runtime.go)
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS totp_device (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  secret VARCHAR(255) NOT NULL,
  confirmed TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_totp_device_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS backup_code (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  code VARCHAR(64) NOT NULL,
  used TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_backup_code_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- OAuth login linking (api/v2/auth/oauth_helpers.go)
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oauth_user_link (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  provider VARCHAR(20) NOT NULL,
  provider_id VARCHAR(100) NOT NULL,
  email VARCHAR(254) NOT NULL,
  access_token LONGTEXT NULL,
  refresh_token LONGTEXT NULL,
  expiry DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_oauth_link (user_id, provider)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- MOSS plagiarism-check result cache (api/v2/moss.go)
-- Distinct from Django's own judge_contestmoss table, which OJ-v2 does
-- not use.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS moss_result (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  primary_submission_id BIGINT UNSIGNED NOT NULL,
  compared_submission_ids LONGTEXT NULL,
  similarity_url LONGTEXT NULL,
  created_at VARCHAR(255) NULL,
  PRIMARY KEY (id),
  KEY idx_moss_result_primary_submission_id (primary_submission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- Comment revision history (models/comment.go CommentRevision)
-- NOTE: uses the judge_ prefix Django reserves for its own migrations.
-- Not currently claimed by any Django migration, but re-check this before
-- ever upgrading the Django schema. Kept as-is per explicit decision
-- (2026-07-19) -- not renamed.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS judge_commentrevision (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  comment_id BIGINT UNSIGNED NOT NULL,
  editor_id BIGINT UNSIGNED NOT NULL,
  `time` DATETIME(6) NOT NULL,
  body LONGTEXT NOT NULL,
  reason VARCHAR(200) NULL,
  PRIMARY KEY (id),
  KEY idx_commentrevision_comment_id (comment_id),
  KEY idx_commentrevision_editor_id (editor_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- Contest clarifications Q&A (models/contest.go ContestClarification)
-- Same judge_ prefix caveat as judge_commentrevision above. Django's own
-- "clarification" concept for contests is ProblemClarification filtered by
-- the contest's problems -- this is a separate, v2-only, contest-scoped
-- Q&A feature. Kept as-is per explicit decision (2026-07-19) -- not renamed.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS judge_contestclarification (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  contest_id BIGINT UNSIGNED NOT NULL,
  question LONGTEXT NOT NULL,
  answer LONGTEXT NULL,
  `time` DATETIME(6) NOT NULL,
  is_answered TINYINT(1) NOT NULL DEFAULT 0,
  is_inlined TINYINT(1) NOT NULL DEFAULT 0,
  author_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (id),
  KEY idx_contestclarification_contest_id (contest_id),
  KEY idx_contestclarification_author_id (author_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- Retained v2 additions to Django-owned tables
-- judge_solution (Django's Solution model: id, problem_id, pdf_url,
-- content, is_public, publish_on, authors only) gains 4 v2-only editorial
-- columns, read/written by api/v2/solution.go. `summary` was found during
-- the schema audit in addition to the 3 columns originally called out
-- (is_official, valid_until, language) -- included here so the feature is
-- actually complete after provisioning; see docs/schema-audit.md section 3.
-- ---------------------------------------------------------------------
ALTER TABLE judge_solution
  ADD COLUMN IF NOT EXISTS is_official TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS valid_until DATETIME(6) NULL,
  ADD COLUMN IF NOT EXISTS summary LONGTEXT NULL,
  ADD COLUMN IF NOT EXISTS `language` VARCHAR(7) NOT NULL DEFAULT 'en';
