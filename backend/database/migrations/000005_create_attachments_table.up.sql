-- 创建附件表
CREATE TABLE IF NOT EXISTS attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id INTEGER NOT NULL,
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

-- 创建附件表索引
CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_email_type ON attachments(email_id, content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_attachments_filename ON attachments(filename);
CREATE INDEX IF NOT EXISTS idx_attachments_is_inline ON attachments(is_inline);

-- 创建更新时间触发器
CREATE TRIGGER IF NOT EXISTS update_attachments_updated_at 
    AFTER UPDATE ON attachments
    FOR EACH ROW
BEGIN
    UPDATE attachments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
