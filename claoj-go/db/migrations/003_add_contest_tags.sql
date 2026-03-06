-- Migration 003: Add contest tags support (Task #23)
-- Creates judge_contesttag table and judge_contest_tags join table

-- Create judge_contesttag table
CREATE TABLE IF NOT EXISTS judge_contesttag (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE COMMENT 'Tag name',
    color VARCHAR(7) NOT NULL COMMENT 'Hex color code (e.g., #FF5733)',
    description LONGTEXT NOT NULL COMMENT 'Tag description'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Contest tags';

-- Create judge_contest_tags join table for many-to-many relationship
CREATE TABLE IF NOT EXISTS judge_contest_tags (
    contest_id INT NOT NULL,
    contesttag_id INT NOT NULL,
    PRIMARY KEY (contest_id, contesttag_id),
    FOREIGN KEY (contest_id) REFERENCES judge_contest(id) ON DELETE CASCADE,
    FOREIGN KEY (contesttag_id) REFERENCES judge_contesttag(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Contest to tag many-to-many relationship';
