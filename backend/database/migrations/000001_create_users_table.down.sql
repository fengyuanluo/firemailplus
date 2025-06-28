-- 删除用户表触发器
DROP TRIGGER IF EXISTS update_users_updated_at;

-- 删除用户表索引
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_is_active;

-- 删除用户表
DROP TABLE IF EXISTS users;
