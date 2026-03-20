-- 回滚邮箱分组 system_key 字段

-- 1. 删除相关索引
DROP INDEX IF EXISTS idx_email_groups_system_unique;
DROP INDEX IF EXISTS idx_email_groups_system_key;

-- 2. SQLite 不支持直接删除列，需要重建表
CREATE TABLE IF NOT EXISTS email_groups_new (
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

-- 3. 复制数据（忽略 system_key）
INSERT INTO email_groups_new (
    id, user_id, name, sort_order, is_default, created_at, updated_at, deleted_at
)
SELECT
    id, user_id, name, sort_order, is_default, created_at, updated_at, deleted_at
FROM email_groups;

-- 4. 删除旧表并重命名新表
DROP TABLE email_groups;
ALTER TABLE email_groups_new RENAME TO email_groups;

-- 5. 重建索引
CREATE INDEX IF NOT EXISTS idx_email_groups_user_id ON email_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_email_groups_sort ON email_groups(user_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_groups_default ON email_groups(user_id, is_default) WHERE is_default = 1;
