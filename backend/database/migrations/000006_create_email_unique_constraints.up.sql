-- 创建邮件唯一约束，防止重复邮件

-- 主要约束：账户内MessageID唯一（排除NULL和空字符串）
CREATE UNIQUE INDEX IF NOT EXISTS idx_emails_account_message_id_unique
ON emails(account_id, message_id)
WHERE message_id IS NOT NULL AND message_id != '';

-- 辅助约束：文件夹内UID唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_emails_account_folder_uid_unique
ON emails(account_id, folder_id, uid)
WHERE folder_id IS NOT NULL;

-- 内容相似性约束：防止相同主题、发件人、日期的邮件重复（用于MessageID为空的情况）
-- 注意：这里使用from_address而不是from，避免SQLite保留字问题
CREATE INDEX IF NOT EXISTS idx_emails_content_similarity
ON emails(account_id, subject, from_address, date)
WHERE message_id IS NULL OR message_id = '';
