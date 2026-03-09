-- Migration: Add WebAuthn support
-- Created: 2026-03-05
-- Description: Adds WebAuthn two-factor authentication support

-- Add is_webauthn_enabled field to judge_profile (if not exists from Django)
-- Note: This field may already exist from Django migration 0105_webauthn.py
ALTER TABLE judge_profile
ADD COLUMN IF NOT EXISTS is_webauthn_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Create WebAuthn credential table (if not exists from Django)
-- Note: This table may already exist from Django migration 0105_webauthn.py
CREATE TABLE IF NOT EXISTS judge_webauthncredential (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    name VARCHAR(100) NOT NULL,
    cred_id VARCHAR(255) NOT NULL UNIQUE,
    public_key LONGTEXT NOT NULL,
    counter BIGINT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES judge_profile(id) ON DELETE CASCADE,
    INDEX idx_webauthn_user_id (user_id),
    INDEX idx_webauthn_cred_id (cred_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
