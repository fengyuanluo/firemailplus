-- 优化数据库索引设计
-- 这个迁移添加了更多针对常用查询模式的复合索引

-- 邮件表的高级复合索引
-- 用于邮件列表查询的优化索引
CREATE INDEX IF NOT EXISTS idx_emails_account_folder_date ON emails(account_id, folder_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_account_read_date ON emails(account_id, is_read, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_account_starred_date ON emails(account_id, is_starred, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_account_important_date ON emails(account_id, is_important, date DESC);

-- 用于搜索的索引
CREATE INDEX IF NOT EXISTS idx_emails_account_subject ON emails(account_id, subject);
CREATE INDEX IF NOT EXISTS idx_emails_account_from ON emails(account_id, from_address);

-- 用于同步的索引
CREATE INDEX IF NOT EXISTS idx_emails_account_uid_folder ON emails(account_id, uid, folder_id);
CREATE INDEX IF NOT EXISTS idx_emails_folder_uid ON emails(folder_id, uid);

-- 用于软删除查询的索引
CREATE INDEX IF NOT EXISTS idx_emails_account_deleted ON emails(account_id, is_deleted, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_deleted_date ON emails(is_deleted, date DESC);

-- 邮件账户表的优化索引
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_active ON email_accounts(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_provider ON email_accounts(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_email_accounts_sync_status ON email_accounts(sync_status);

-- 文件夹表的优化索引
CREATE INDEX IF NOT EXISTS idx_folders_account_type ON folders(account_id, type);
CREATE INDEX IF NOT EXISTS idx_folders_account_selectable ON folders(account_id, is_selectable);
CREATE INDEX IF NOT EXISTS idx_folders_account_path ON folders(account_id, path);

-- 附件表的优化索引
CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_type ON attachments(content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_size ON attachments(size);

-- 邮件模板表的索引（如果存在）
CREATE INDEX IF NOT EXISTS idx_email_templates_user_id ON email_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_email_templates_name ON email_templates(name);

-- 草稿表的索引（如果存在）
CREATE INDEX IF NOT EXISTS idx_drafts_user_id ON drafts(user_id);
CREATE INDEX IF NOT EXISTS idx_drafts_account_id ON drafts(account_id);

-- OAuth2状态表的索引（如果存在）
CREATE INDEX IF NOT EXISTS idx_oauth2_states_state ON oauth2_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth2_states_expires_at ON oauth2_states(expires_at);

-- 为了提高查询性能，添加一些覆盖索引
-- 这些索引包含了查询中需要的所有列，避免回表查询

-- 邮件列表查询的覆盖索引
CREATE INDEX IF NOT EXISTS idx_emails_list_cover ON emails(
    account_id, 
    folder_id, 
    is_deleted, 
    date DESC, 
    id, 
    subject, 
    from_address, 
    is_read, 
    is_starred, 
    is_important, 
    has_attachment
);

-- 邮件统计查询的覆盖索引
CREATE INDEX IF NOT EXISTS idx_emails_stats_cover ON emails(
    account_id, 
    folder_id, 
    is_read, 
    is_deleted
);

-- 同步查询的覆盖索引
CREATE INDEX IF NOT EXISTS idx_emails_sync_cover ON emails(
    account_id, 
    folder_id, 
    uid, 
    message_id, 
    is_deleted
);
