-- 回滚邮箱分组：删除分组表并移除账户分组字段

-- 1. 删除触发器（如果存在）
DROP TRIGGER IF EXISTS update_email_accounts_updated_at;

-- 2. 使用临时表移除group_id列
CREATE TABLE IF NOT EXISTS email_accounts_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    auth_method VARCHAR(20) NOT NULL,

    -- IMAP配置
    imap_host VARCHAR(100),
    imap_port INTEGER DEFAULT 993,
    imap_security VARCHAR(20) DEFAULT 'SSL',

    -- SMTP配置
    smtp_host VARCHAR(100),
    smtp_port INTEGER DEFAULT 587,
    smtp_security VARCHAR(20) DEFAULT 'STARTTLS',

    -- 认证信息
    username VARCHAR(100),
    password VARCHAR(255),
    oauth2_token TEXT,

    -- 状态信息
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_sync_at DATETIME,
    sync_status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,

    -- 统计信息
    total_emails INTEGER DEFAULT 0,
    unread_emails INTEGER DEFAULT 0,

    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 3. 复制数据（忽略group_id）
INSERT INTO email_accounts_new (
    id, user_id, name, email, provider, auth_method,
    imap_host, imap_port, imap_security,
    smtp_host, smtp_port, smtp_security,
    username, password, oauth2_token,
    is_active, last_sync_at, sync_status, error_message,
    total_emails, unread_emails,
    created_at, updated_at, deleted_at
)
SELECT
    id, user_id, name, email, provider, auth_method,
    imap_host, imap_port, imap_security,
    smtp_host, smtp_port, smtp_security,
    username, password, oauth2_token,
    is_active, last_sync_at, sync_status, error_message,
    total_emails, unread_emails,
    created_at, updated_at, deleted_at
FROM email_accounts;

-- 4. 删除旧表并重命名新表
DROP TABLE email_accounts;
ALTER TABLE email_accounts_new RENAME TO email_accounts;

-- 5. 重建索引
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_id ON email_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_provider ON email_accounts(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_email_accounts_email ON email_accounts(email);
CREATE INDEX IF NOT EXISTS idx_email_accounts_deleted_at ON email_accounts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_email_accounts_is_active ON email_accounts(is_active);
CREATE INDEX IF NOT EXISTS idx_email_accounts_sync_status ON email_accounts(sync_status);

-- 6. 重建更新时间触发器
CREATE TRIGGER IF NOT EXISTS update_email_accounts_updated_at 
    AFTER UPDATE ON email_accounts
    FOR EACH ROW
BEGIN
    UPDATE email_accounts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 7. 删除邮箱分组表
DROP TABLE IF EXISTS email_groups;
