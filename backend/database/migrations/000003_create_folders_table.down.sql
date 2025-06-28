-- 删除文件夹表触发器
DROP TRIGGER IF EXISTS update_folders_updated_at;

-- 删除文件夹表索引
DROP INDEX IF EXISTS idx_folders_account_id;
DROP INDEX IF EXISTS idx_folders_account_type;
DROP INDEX IF EXISTS idx_folders_account_parent;
DROP INDEX IF EXISTS idx_folders_parent_id;
DROP INDEX IF EXISTS idx_folders_deleted_at;
DROP INDEX IF EXISTS idx_folders_type;
DROP INDEX IF EXISTS idx_folders_path;

-- 删除文件夹表
DROP TABLE IF EXISTS folders;
