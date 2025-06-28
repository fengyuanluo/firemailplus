-- 创建发送队列表
CREATE TABLE IF NOT EXISTS send_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    -- 基本信息
    send_id VARCHAR(100) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    account_id INTEGER NOT NULL,
    
    -- 邮件内容 (JSON格式)
    email_data TEXT NOT NULL,
    
    -- 发送设置
    scheduled_at DATETIME,
    priority INTEGER DEFAULT 5,
    
    -- 状态
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- 错误信息
    last_error TEXT,
    last_attempt DATETIME,
    next_attempt DATETIME,
    
    -- 索引
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (account_id) REFERENCES email_accounts(id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_send_queue_user_id ON send_queue(user_id);
CREATE INDEX IF NOT EXISTS idx_send_queue_account_id ON send_queue(account_id);
CREATE INDEX IF NOT EXISTS idx_send_queue_status ON send_queue(status);
CREATE INDEX IF NOT EXISTS idx_send_queue_scheduled_at ON send_queue(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_send_queue_next_attempt ON send_queue(next_attempt);
CREATE INDEX IF NOT EXISTS idx_send_queue_deleted_at ON send_queue(deleted_at);
