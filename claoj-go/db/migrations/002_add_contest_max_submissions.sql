-- Migration 002: Add max_submissions column to judge_contest table
-- This adds per-contest total submission limit support (Task #28)

-- Add max_submissions column to judge_contest
ALTER TABLE judge_contest
ADD COLUMN max_submissions INT NULL COMMENT 'Maximum total submissions per user in this contest (NULL = no limit)';
