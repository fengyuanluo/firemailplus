-- 为附件表添加user_id字段，用于临时附件的权限检查
-- 临时附件（email_id为NULL）需要通过user_id来确定所有者

-- 1. 添加user_id列
ALTER TABLE attachments ADD COLUMN user_id INTEGER;

-- 2. 为user_id字段创建索引
CREATE INDEX IF NOT EXISTS idx_attachments_user_id ON attachments(user_id);

-- 3. 为临时附件权限检查创建复合索引
CREATE INDEX IF NOT EXISTS idx_attachments_temp_permission ON attachments(user_id, email_id) WHERE email_id IS NULL;

-- 4. 添加外键约束（如果需要的话，SQLite默认不强制外键）
-- 注意：在生产环境中可能需要根据实际情况决定是否添加外键约束
