-- 创建OAuth2状态表
CREATE TABLE IF NOT EXISTS oauth2_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    state VARCHAR(128) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    provider VARCHAR(50) NOT NULL,
    metadata TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_oauth2_states_state ON oauth2_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth2_states_user_id ON oauth2_states(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_states_expires_at ON oauth2_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth2_states_provider ON oauth2_states(provider);
