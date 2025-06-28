-- 回滚：将email_id字段改回NOT NULL约束
-- 注意：这个回滚操作会删除所有email_id为NULL的附件记录

-- 1. 删除email_id为NULL的记录（临时上传但未关联的附件）
DELETE FROM attachments WHERE email_id IS NULL;

-- 2. 创建新表结构（恢复NOT NULL约束）
CREATE TABLE IF NOT EXISTS attachments_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id INTEGER NOT NULL,  -- 恢复NOT NULL约束
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
    
    -- 外键约束
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

-- 3. 复制现有数据（只复制email_id不为NULL的记录）
INSERT INTO attachments_new (id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, created_at, updated_at, deleted_at)
SELECT id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, created_at, updated_at, deleted_at
FROM attachments
WHERE email_id IS NOT NULL;

-- 4. 删除旧表
DROP TABLE attachments;

-- 5. 重命名新表
ALTER TABLE attachments_new RENAME TO attachments;

-- 6. 重建索引
CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_email_type ON attachments(email_id, content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_attachments_filename ON attachments(filename);
CREATE INDEX IF NOT EXISTS idx_attachments_is_inline ON attachments(is_inline);

-- 7. 重建触发器
CREATE TRIGGER IF NOT EXISTS update_attachments_updated_at 
    AFTER UPDATE ON attachments
    FOR EACH ROW
BEGIN
    UPDATE attachments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
