-- 删除附件表触发器
DROP TRIGGER IF EXISTS update_attachments_updated_at;

-- 删除附件表索引
DROP INDEX IF EXISTS idx_attachments_email_id;
DROP INDEX IF EXISTS idx_attachments_email_type;
DROP INDEX IF EXISTS idx_attachments_content_id;
DROP INDEX IF EXISTS idx_attachments_deleted_at;
DROP INDEX IF EXISTS idx_attachments_filename;
DROP INDEX IF EXISTS idx_attachments_is_inline;

-- 删除附件表
DROP TABLE IF EXISTS attachments;
