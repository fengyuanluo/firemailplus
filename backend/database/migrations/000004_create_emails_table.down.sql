-- 删除邮件表触发器
DROP TRIGGER IF EXISTS update_emails_updated_at;

-- 删除邮件表索引
DROP INDEX IF EXISTS idx_emails_account_id;
DROP INDEX IF EXISTS idx_emails_folder_id;
DROP INDEX IF EXISTS idx_emails_message_id;
DROP INDEX IF EXISTS idx_emails_uid;
DROP INDEX IF EXISTS idx_emails_date;
DROP INDEX IF EXISTS idx_emails_deleted_at;
DROP INDEX IF EXISTS idx_emails_account_folder;
DROP INDEX IF EXISTS idx_emails_account_date;
DROP INDEX IF EXISTS idx_emails_account_read;
DROP INDEX IF EXISTS idx_emails_message_uid;
DROP INDEX IF EXISTS idx_emails_is_read;
DROP INDEX IF EXISTS idx_emails_is_starred;
DROP INDEX IF EXISTS idx_emails_is_deleted;
DROP INDEX IF EXISTS idx_emails_is_draft;
DROP INDEX IF EXISTS idx_emails_is_sent;
DROP INDEX IF EXISTS idx_emails_has_attachment;

-- 删除邮件表
DROP TABLE IF EXISTS emails;
