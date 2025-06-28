-- 回滚：移除附件表的user_id字段

-- 1. 删除相关索引
DROP INDEX IF EXISTS idx_attachments_temp_permission;
DROP INDEX IF EXISTS idx_attachments_user_id;

-- 2. 删除user_id列
ALTER TABLE attachments DROP COLUMN user_id;
