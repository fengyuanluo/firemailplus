-- 回滚：删除邮箱账户分组表及账户表中的分组/排序字段

-- 1. 删除邮箱账户表上的新索引
DROP INDEX IF EXISTS idx_email_accounts_group_id;
DROP INDEX IF EXISTS idx_email_accounts_user_sort;

-- 2. 移除邮箱账户表的分组/排序字段（通过重建表实现）
CREATE TABLE IF NOT EXISTS email_accounts_temp (
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

    -- 代理配置
    proxy_url VARCHAR(500) DEFAULT '',

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

INSERT INTO email_accounts_temp (
    id, user_id, name, email, provider, auth_method,
    imap_host, imap_port, imap_security,
    smtp_host, smtp_port, smtp_security,
    username, password, oauth2_token,
    proxy_url,
    is_active, last_sync_at, sync_status, error_message,
    total_emails, unread_emails,
    created_at, updated_at, deleted_at
)
SELECT
    id, user_id, name, email, provider, auth_method,
    imap_host, imap_port, imap_security,
    smtp_host, smtp_port, smtp_security,
    username, password, oauth2_token,
    proxy_url,
    is_active, last_sync_at, sync_status, error_message,
    total_emails, unread_emails,
    created_at, updated_at, deleted_at
FROM email_accounts;

DROP TABLE email_accounts;
ALTER TABLE email_accounts_temp RENAME TO email_accounts;

-- 3. 重建原有索引
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_id ON email_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_provider ON email_accounts(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_email_accounts_email ON email_accounts(email);
CREATE INDEX IF NOT EXISTS idx_email_accounts_deleted_at ON email_accounts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_email_accounts_is_active ON email_accounts(is_active);
CREATE INDEX IF NOT EXISTS idx_email_accounts_sync_status ON email_accounts(sync_status);
CREATE INDEX IF NOT EXISTS idx_email_accounts_proxy_url ON email_accounts(proxy_url) WHERE proxy_url != '';

-- 4. 重建更新时间触发器
CREATE TRIGGER IF NOT EXISTS update_email_accounts_updated_at
    AFTER UPDATE ON email_accounts
    FOR EACH ROW
BEGIN
    UPDATE email_accounts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 5. 删除邮箱账户分组表及相关对象
DROP TRIGGER IF EXISTS update_email_account_groups_updated_at;
DROP INDEX IF EXISTS idx_email_account_groups_user_sort;
DROP INDEX IF EXISTS idx_email_account_groups_user_id;
DROP INDEX IF EXISTS idx_email_account_groups_deleted_at;
DROP TABLE IF EXISTS email_account_groups;
