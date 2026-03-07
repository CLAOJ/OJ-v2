-- Create refresh_token table for JWT token rotation and revocation
CREATE TABLE IF NOT EXISTS refresh_token (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    token VARCHAR(512) NOT NULL,
    revoked TINYINT(1) NOT NULL DEFAULT 0,
    revoked_at TIMESTAMP NULL DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    user_agent VARCHAR(512) DEFAULT NULL,
    client_ip VARCHAR(39) DEFAULT NULL,
    family_id VARCHAR(64) NOT NULL,
    UNIQUE KEY idx_token (token),
    KEY idx_user_id (user_id),
    KEY idx_created_at (created_at),
    KEY idx_expires_at (expires_at),
    KEY idx_family_id (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
