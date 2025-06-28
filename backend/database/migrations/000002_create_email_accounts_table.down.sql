-- 删除邮件账户表触发器
DROP TRIGGER IF EXISTS update_email_accounts_updated_at;

-- 删除邮件账户表索引
DROP INDEX IF EXISTS idx_email_accounts_user_id;
DROP INDEX IF EXISTS idx_email_accounts_user_provider;
DROP INDEX IF EXISTS idx_email_accounts_email;
DROP INDEX IF EXISTS idx_email_accounts_deleted_at;
DROP INDEX IF EXISTS idx_email_accounts_is_active;
DROP INDEX IF EXISTS idx_email_accounts_sync_status;

-- 删除邮件账户表
DROP TABLE IF EXISTS email_accounts;
