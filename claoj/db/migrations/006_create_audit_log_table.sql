-- Migration: 006_create_audit_log_table
-- Description: Creates audit log table for tracking admin actions
-- Date: 2026-03-06

CREATE TABLE IF NOT EXISTS audit_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    username VARCHAR(150) NOT NULL,
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100),
    ip_address VARCHAR(45),
    user_agent VARCHAR(500),
    details TEXT,
    status VARCHAR(20) DEFAULT 'success',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_audit_user_id (user_id),
    INDEX idx_audit_action (action),
    INDEX idx_audit_resource (resource),
    INDEX idx_audit_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
