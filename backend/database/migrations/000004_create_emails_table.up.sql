-- 创建邮件表
CREATE TABLE IF NOT EXISTS emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    folder_id INTEGER,
    message_id VARCHAR(255) NOT NULL,
    uid INTEGER NOT NULL,
    
    -- 邮件头信息
    subject VARCHAR(500),
    from_address VARCHAR(255),
    to_addresses TEXT,
    cc_addresses TEXT,
    bcc_addresses TEXT,
    reply_to VARCHAR(255),
    date DATETIME,
    
    -- 邮件内容
    text_body TEXT,
    html_body TEXT,
    
    -- 邮件状态
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    is_important BOOLEAN NOT NULL DEFAULT false,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    is_draft BOOLEAN NOT NULL DEFAULT false,
    is_sent BOOLEAN NOT NULL DEFAULT false,
    
    -- 邮件大小和附件信息
    size INTEGER DEFAULT 0,
    has_attachment BOOLEAN NOT NULL DEFAULT false,
    
    -- 邮件标签和分类
    labels TEXT,
    priority VARCHAR(20) DEFAULT 'normal',
    
    -- 同步信息
    synced_at DATETIME,
    
    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 外键约束
    FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE SET NULL
);

-- 创建邮件表基础索引
CREATE INDEX IF NOT EXISTS idx_emails_account_id ON emails(account_id);
CREATE INDEX IF NOT EXISTS idx_emails_folder_id ON emails(folder_id);
CREATE INDEX IF NOT EXISTS idx_emails_message_id ON emails(message_id);
CREATE INDEX IF NOT EXISTS idx_emails_uid ON emails(uid);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_deleted_at ON emails(deleted_at);

-- 创建邮件表复合索引
CREATE INDEX IF NOT EXISTS idx_emails_account_folder ON emails(account_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_emails_account_date ON emails(account_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_account_read ON emails(account_id, is_read);
CREATE INDEX IF NOT EXISTS idx_emails_message_uid ON emails(message_id, uid);

-- 创建邮件状态索引
CREATE INDEX IF NOT EXISTS idx_emails_is_read ON emails(is_read);
CREATE INDEX IF NOT EXISTS idx_emails_is_starred ON emails(is_starred);
CREATE INDEX IF NOT EXISTS idx_emails_is_deleted ON emails(is_deleted);
CREATE INDEX IF NOT EXISTS idx_emails_is_draft ON emails(is_draft);
CREATE INDEX IF NOT EXISTS idx_emails_is_sent ON emails(is_sent);
CREATE INDEX IF NOT EXISTS idx_emails_has_attachment ON emails(has_attachment);

-- 创建更新时间触发器
CREATE TRIGGER IF NOT EXISTS update_emails_updated_at 
    AFTER UPDATE ON emails
    FOR EACH ROW
BEGIN
    UPDATE emails SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
