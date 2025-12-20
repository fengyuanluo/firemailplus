-- 创建邮箱分组表
CREATE TABLE IF NOT EXISTS email_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 基础索引
CREATE INDEX IF NOT EXISTS idx_email_groups_user_id ON email_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_email_groups_sort ON email_groups(user_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_groups_default ON email_groups(user_id, is_default) WHERE is_default = 1;

-- 为邮件账户增加分组字段
ALTER TABLE email_accounts ADD COLUMN group_id INTEGER REFERENCES email_groups(id);

-- 为现有用户创建默认分组
INSERT INTO email_groups (user_id, name, sort_order, is_default, created_at, updated_at)
SELECT id, '未分组', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM users
WHERE id NOT IN (SELECT user_id FROM email_groups WHERE is_default = 1);

-- 将已有账户归入默认分组
UPDATE email_accounts
SET group_id = (
  SELECT id FROM email_groups eg WHERE eg.user_id = email_accounts.user_id AND eg.is_default = 1
)
WHERE group_id IS NULL;
