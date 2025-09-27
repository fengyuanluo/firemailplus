-- 创建邮箱账户分组表并为账户表添加分组/排序支持

-- 创建邮箱账户分组表
CREATE TABLE IF NOT EXISTS email_account_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_email_account_groups_user_id ON email_account_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_email_account_groups_user_sort ON email_account_groups(user_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_email_account_groups_deleted_at ON email_account_groups(deleted_at);

-- 创建更新时间触发器
CREATE TRIGGER IF NOT EXISTS update_email_account_groups_updated_at
    AFTER UPDATE ON email_account_groups
    FOR EACH ROW
BEGIN
    UPDATE email_account_groups SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 为邮箱账户表新增分组及排序字段
ALTER TABLE email_accounts ADD COLUMN group_id INTEGER REFERENCES email_account_groups(id) ON DELETE SET NULL;
ALTER TABLE email_accounts ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

-- 为新字段创建索引
CREATE INDEX IF NOT EXISTS idx_email_accounts_group_id ON email_accounts(group_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_sort ON email_accounts(user_id, sort_order);
