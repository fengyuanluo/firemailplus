-- 移除文件夹同步字段
ALTER TABLE folders DROP COLUMN uid_validity;
ALTER TABLE folders DROP COLUMN uid_next;
