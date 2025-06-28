-- 为附件表添加encoding字段，用于存储附件的传输编码信息
-- 这将解决附件下载时编码信息丢失导致文件损坏的问题

-- 1. 添加encoding列
ALTER TABLE attachments ADD COLUMN encoding VARCHAR(50) NOT NULL DEFAULT '7bit';

-- 2. 为encoding字段创建索引（用于查询优化）
CREATE INDEX IF NOT EXISTS idx_attachments_encoding ON attachments(encoding);

-- 3. 为常见的编码类型组合创建复合索引（可选，用于性能优化）
CREATE INDEX IF NOT EXISTS idx_attachments_content_encoding ON attachments(content_type, encoding);

-- 4. 更新现有记录的编码信息
-- 对于已存在的附件，根据content_type推断可能的编码类型
-- 大多数二进制文件（图片、文档等）通常使用base64编码
-- 文本文件通常使用7bit或quoted-printable编码

UPDATE attachments SET encoding = 'base64' 
WHERE content_type IN (
    'image/jpeg', 'image/png', 'image/gif', 'image/bmp', 'image/webp',
    'application/pdf', 'application/msword', 'application/vnd.ms-excel',
    'application/vnd.ms-powerpoint', 'application/zip', 'application/rar',
    'application/octet-stream', 'video/mp4', 'audio/mpeg'
) AND encoding = '7bit';

UPDATE attachments SET encoding = 'quoted-printable'
WHERE content_type LIKE 'text/%' 
AND content_type NOT IN ('text/plain', 'text/html')
AND encoding = '7bit';

-- 注意：这些更新只是基于常见模式的推断
-- 实际的编码信息在原始邮件解析时确定，新的附件将正确存储编码信息
