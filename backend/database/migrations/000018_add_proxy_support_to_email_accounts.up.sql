-- 为邮件账户表添加代理支持
-- 使用简化的URL格式，支持HTTP代理和SOCKS5代理

-- 添加代理URL字段
ALTER TABLE email_accounts ADD COLUMN proxy_url VARCHAR(500) DEFAULT '';

-- 为代理URL字段创建索引（用于查询优化）
CREATE INDEX IF NOT EXISTS idx_email_accounts_proxy_url ON email_accounts(proxy_url) WHERE proxy_url != '';

-- 注释说明：
-- proxy_url 支持标准代理URL格式：
-- - HTTP代理：http://proxy.company.com:8080
-- - HTTP代理(带认证)：http://username:password@proxy.company.com:8080
-- - SOCKS5代理：socks5://proxy.company.com:1080
-- - SOCKS5代理(带认证)：socks5://username:password@proxy.company.com:1080
-- - 空字符串表示不使用代理
-- 现有邮箱账户的proxy_url默认为空字符串，确保向后兼容
