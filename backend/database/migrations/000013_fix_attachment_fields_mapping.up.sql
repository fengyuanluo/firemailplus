-- 修复附件表字段映射语义错误
-- 添加正确的is_downloaded字段，修正is_inline字段的语义

-- 1. 添加新的is_downloaded列
ALTER TABLE attachments ADD COLUMN is_downloaded BOOLEAN NOT NULL DEFAULT false;

-- 2. 将现有is_inline列的数据迁移到is_downloaded列
-- 因为当前is_inline列实际存储的是"是否已下载"的信息
UPDATE attachments SET is_downloaded = is_inline;

-- 3. 重新计算is_inline列的正确值
-- is_inline应该表示是否为内联附件，基于disposition和content_id判断
UPDATE attachments SET is_inline = (
    disposition = 'inline' OR 
    (content_id IS NOT NULL AND content_id != '')
);

-- 4. 为新字段创建索引
CREATE INDEX IF NOT EXISTS idx_attachments_is_downloaded ON attachments(is_downloaded);

-- 5. 更新现有索引的注释（SQLite不支持注释，但保留用于文档）
-- idx_attachments_is_inline 现在正确表示内联附件索引
