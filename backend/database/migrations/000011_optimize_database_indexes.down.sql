-- 回滚索引优化
-- 删除在000011_optimize_database_indexes.up.sql中创建的索引

-- 删除邮件表的高级复合索引
DROP INDEX IF EXISTS idx_emails_account_folder_date;
DROP INDEX IF EXISTS idx_emails_account_read_date;
DROP INDEX IF EXISTS idx_emails_account_starred_date;
DROP INDEX IF EXISTS idx_emails_account_important_date;

-- 删除搜索索引
DROP INDEX IF EXISTS idx_emails_account_subject;
DROP INDEX IF EXISTS idx_emails_account_from;

-- 删除同步索引
DROP INDEX IF EXISTS idx_emails_account_uid_folder;
DROP INDEX IF EXISTS idx_emails_folder_uid;

-- 删除软删除索引
DROP INDEX IF EXISTS idx_emails_account_deleted;
DROP INDEX IF EXISTS idx_emails_deleted_date;

-- 删除邮件账户表索引
DROP INDEX IF EXISTS idx_email_accounts_user_active;
DROP INDEX IF EXISTS idx_email_accounts_user_provider;
DROP INDEX IF EXISTS idx_email_accounts_sync_status;

-- 删除文件夹表索引
DROP INDEX IF EXISTS idx_folders_account_type;
DROP INDEX IF EXISTS idx_folders_account_selectable;
DROP INDEX IF EXISTS idx_folders_account_path;

-- 删除附件表索引
DROP INDEX IF EXISTS idx_attachments_email_id;
DROP INDEX IF EXISTS idx_attachments_content_type;
DROP INDEX IF EXISTS idx_attachments_size;

-- 删除邮件模板表索引
DROP INDEX IF EXISTS idx_email_templates_user_id;
DROP INDEX IF EXISTS idx_email_templates_name;

-- 删除草稿表索引
DROP INDEX IF EXISTS idx_drafts_user_id;
DROP INDEX IF EXISTS idx_drafts_account_id;

-- 删除OAuth2状态表索引
DROP INDEX IF EXISTS idx_oauth2_states_state;
DROP INDEX IF EXISTS idx_oauth2_states_expires_at;

-- 删除覆盖索引
DROP INDEX IF EXISTS idx_emails_list_cover;
DROP INDEX IF EXISTS idx_emails_stats_cover;
DROP INDEX IF EXISTS idx_emails_sync_cover;
