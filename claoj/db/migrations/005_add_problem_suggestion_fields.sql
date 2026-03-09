-- Migration: Add problem suggestion fields
-- Date: 2026-03-05
-- Description: Add fields to track problem suggestions and approval workflow

-- Add suggestion status column (default 'none' for existing problems)
ALTER TABLE judge_problem
ADD COLUMN suggestion_status VARCHAR(20) NOT NULL DEFAULT 'none'
COMMENT 'Suggestion status: none, pending, approved, rejected';

-- Add suggestion notes column for admin review notes
ALTER TABLE judge_problem
ADD COLUMN suggestion_notes LONGTEXT NOT NULL DEFAULT ''
COMMENT 'Admin notes for suggestion review';

-- Add suggestion reviewed at timestamp
ALTER TABLE judge_problem
ADD COLUMN suggestion_reviewed_at DATETIME NULL
COMMENT 'When the suggestion was reviewed';

-- Add suggestion reviewed by foreign key
ALTER TABLE judge_problem
ADD COLUMN suggestion_reviewed_by_id INT UNSIGNED NULL
COMMENT 'Profile ID of the admin who reviewed the suggestion';

-- Add index for filtering pending suggestions
CREATE INDEX idx_problem_suggestion_status ON judge_problem(suggestion_status);

-- Add foreign key constraint for reviewed by
ALTER TABLE judge_problem
ADD CONSTRAINT fk_problem_suggestion_reviewed_by
FOREIGN KEY (suggestion_reviewed_by_id) REFERENCES judge_profile(id)
ON DELETE SET NULL;
