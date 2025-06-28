-- 回滚：删除附件表的encoding字段

-- 1. 删除相关索引
DROP INDEX IF EXISTS idx_attachments_content_encoding;
DROP INDEX IF EXISTS idx_attachments_encoding;

-- 2. 删除encoding列
-- 注意：SQLite不支持直接删除列，需要重建表
CREATE TABLE IF NOT EXISTS attachments_temp (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id INTEGER,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100),
    size INTEGER DEFAULT 0,
    content_id VARCHAR(255),
    disposition VARCHAR(50),
    
    -- 文件存储信息
    file_path VARCHAR(500),
    is_inline BOOLEAN NOT NULL DEFAULT false,
    is_downloaded BOOLEAN NOT NULL DEFAULT false,
    
    -- 用户权限字段
    user_id INTEGER,
    
    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 外键约束
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

-- 3. 复制数据（不包括encoding列）
INSERT INTO attachments_temp (id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, is_downloaded, user_id, created_at, updated_at, deleted_at)
SELECT id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, is_downloaded, user_id, created_at, updated_at, deleted_at
FROM attachments;

-- 4. 删除原表
DROP TABLE attachments;

-- 5. 重命名临时表
ALTER TABLE attachments_temp RENAME TO attachments;

-- 6. 重建所有索引
CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_email_type ON attachments(email_id, content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_attachments_filename ON attachments(filename);
CREATE INDEX IF NOT EXISTS idx_attachments_is_inline ON attachments(is_inline);
CREATE INDEX IF NOT EXISTS idx_attachments_is_downloaded ON attachments(is_downloaded);
CREATE INDEX IF NOT EXISTS idx_attachments_user_id ON attachments(user_id);
CREATE INDEX IF NOT EXISTS idx_attachments_temp_permission ON attachments(user_id, email_id) WHERE email_id IS NULL;

-- 7. 重建触发器
CREATE TRIGGER IF NOT EXISTS update_attachments_updated_at 
    AFTER UPDATE ON attachments
    FOR EACH ROW
BEGIN
    UPDATE attachments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
