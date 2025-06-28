-- 创建草稿表
CREATE TABLE IF NOT EXISTS drafts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    account_id INTEGER NOT NULL,
    subject VARCHAR(500),
    
    -- 收件人信息 (JSON格式)
    to_addresses TEXT,
    cc_addresses TEXT,
    bcc_addresses TEXT,
    
    -- 邮件内容
    text_body TEXT,
    html_body TEXT,
    
    -- 附件信息 (JSON格式的附件ID列表)
    attachment_ids TEXT,
    
    -- 元数据
    priority VARCHAR(20) DEFAULT 'normal',
    is_template BOOLEAN DEFAULT false,
    template_name VARCHAR(100),
    last_edited_at DATETIME,
    
    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_drafts_user_id ON drafts(user_id);
CREATE INDEX IF NOT EXISTS idx_drafts_account_id ON drafts(account_id);
CREATE INDEX IF NOT EXISTS idx_drafts_is_template ON drafts(is_template);
CREATE INDEX IF NOT EXISTS idx_drafts_created_at ON drafts(created_at);
CREATE INDEX IF NOT EXISTS idx_drafts_last_edited_at ON drafts(last_edited_at);
