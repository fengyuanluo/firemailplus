-- 创建文件夹表
CREATE TABLE IF NOT EXISTS folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    type VARCHAR(20) NOT NULL,
    parent_id INTEGER,
    path VARCHAR(500),
    delimiter VARCHAR(10),
    
    -- 文件夹属性
    is_selectable BOOLEAN NOT NULL DEFAULT true,
    is_subscribed BOOLEAN NOT NULL DEFAULT true,
    
    -- 统计信息
    total_emails INTEGER DEFAULT 0,
    unread_emails INTEGER DEFAULT 0,
    
    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 外键约束
    FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE SET NULL
);

-- 创建文件夹表索引
CREATE INDEX IF NOT EXISTS idx_folders_account_id ON folders(account_id);
CREATE INDEX IF NOT EXISTS idx_folders_account_type ON folders(account_id, type);
CREATE INDEX IF NOT EXISTS idx_folders_account_parent ON folders(account_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_folders_deleted_at ON folders(deleted_at);
CREATE INDEX IF NOT EXISTS idx_folders_type ON folders(type);
CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);

-- 创建更新时间触发器
CREATE TRIGGER IF NOT EXISTS update_folders_updated_at 
    AFTER UPDATE ON folders
    FOR EACH ROW
BEGIN
    UPDATE folders SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
