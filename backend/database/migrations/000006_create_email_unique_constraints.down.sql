-- 删除邮件唯一约束

-- 删除内容相似性约束
DROP INDEX IF EXISTS idx_emails_content_similarity;

-- 删除文件夹内UID唯一约束
DROP INDEX IF EXISTS idx_emails_account_folder_uid_unique;

-- 删除账户内MessageID唯一约束
DROP INDEX IF EXISTS idx_emails_account_message_id_unique;
