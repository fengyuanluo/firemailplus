-- 为附件表添加part_id字段，用于存储IMAP PartID信息
-- 这是修复附件下载失败的关键字段

-- 1. 添加part_id列
ALTER TABLE attachments ADD COLUMN part_id VARCHAR(50);

-- 2. 为part_id字段创建索引
CREATE INDEX IF NOT EXISTS idx_attachments_part_id ON attachments(part_id);

-- 3. 为email_id和part_id创建复合索引（用于IMAP查询优化）
CREATE INDEX IF NOT EXISTS idx_attachments_email_part ON attachments(email_id, part_id);

-- 注意：现有记录的part_id将为NULL，需要重新同步邮件来获取正确的PartID
