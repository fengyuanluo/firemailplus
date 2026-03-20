-- 为邮箱分组增加系统占位标记
ALTER TABLE email_groups ADD COLUMN system_key VARCHAR(50);

-- 基础索引
CREATE INDEX IF NOT EXISTS idx_email_groups_system_key ON email_groups(user_id, system_key);

-- 每个用户同一个 system_key 只能有一条分组记录
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_groups_system_unique
ON email_groups(user_id, system_key)
WHERE system_key IS NOT NULL;
