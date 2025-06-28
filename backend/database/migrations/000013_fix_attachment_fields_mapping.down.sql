-- 回滚附件表字段映射修复
-- 恢复到修复前的状态

-- 1. 将is_downloaded的值恢复到is_inline列
-- 这样可以恢复到修复前的错误映射状态
UPDATE attachments SET is_inline = is_downloaded;

-- 2. 删除is_downloaded列的索引
DROP INDEX IF EXISTS idx_attachments_is_downloaded;

-- 3. 删除is_downloaded列
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
    
    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 外键约束
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

-- 4. 复制数据（不包括is_downloaded列）
INSERT INTO attachments_temp (id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, created_at, updated_at, deleted_at)
SELECT id, email_id, filename, content_type, size, content_id, disposition, file_path, is_inline, created_at, updated_at, deleted_at
FROM attachments;

-- 5. 删除原表
DROP TABLE attachments;

-- 6. 重命名临时表
ALTER TABLE attachments_temp RENAME TO attachments;

-- 7. 重建索引
CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_email_type ON attachments(email_id, content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_attachments_filename ON attachments(filename);
CREATE INDEX IF NOT EXISTS idx_attachments_is_inline ON attachments(is_inline);

-- 8. 重建触发器
CREATE TRIGGER IF NOT EXISTS update_attachments_updated_at 
    AFTER UPDATE ON attachments
    FOR EACH ROW
BEGIN
    UPDATE attachments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
