-- 修改附件表的email_id字段允许为空
-- 这样可以支持临时上传的附件，发送邮件时再关联到邮件记录

-- 由于SQLite不支持直接修改列约束，需要重建表
-- 1. 创建新表结构
CREATE TABLE IF NOT EXISTS attachments_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id INTEGER,  -- 移除NOT NULL约束
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100),
    size INTEGER DEFAULT 0,
    content_id VARCHAR(255),
    disposition VARCHAR(50),
    
    -- 文件存储信息
    file_path VARCHAR(500),
    is_inline BOOLEAN NOT NULL DEFAULT false,
    
    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 外键约束（允许NULL值）
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

-- 2. 复制现有数据
INSERT INTO attachments_new (id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, created_at, updated_at, deleted_at)
SELECT id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, created_at, updated_at, deleted_at
FROM attachments;

-- 3. 删除旧表
DROP TABLE attachments;

-- 4. 重命名新表
ALTER TABLE attachments_new RENAME TO attachments;

-- 5. 重建索引
CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_email_type ON attachments(email_id, content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_attachments_filename ON attachments(filename);
CREATE INDEX IF NOT EXISTS idx_attachments_is_inline ON attachments(is_inline);

-- 6. 重建触发器
CREATE TRIGGER IF NOT EXISTS update_attachments_updated_at 
    AFTER UPDATE ON attachments
    FOR EACH ROW
BEGIN
    UPDATE attachments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
