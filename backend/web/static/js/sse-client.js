/**
 * FireMail SSE客户端
 * 用于接收实时邮件通知和事件
 */
class FireMailSSEClient {
    constructor(options = {}) {
        this.baseUrl = options.baseUrl || '';
        this.token = options.token || '';
        this.clientId = options.clientId || this.generateClientId();
        this.autoReconnect = options.autoReconnect !== false;
        this.reconnectInterval = options.reconnectInterval || 5000;
        this.maxReconnectAttempts = options.maxReconnectAttempts || 10;
        
        this.eventSource = null;
        this.reconnectAttempts = 0;
        this.isConnected = false;
        this.eventHandlers = new Map();
        
        // 绑定方法
        this.onOpen = this.onOpen.bind(this);
        this.onMessage = this.onMessage.bind(this);
        this.onError = this.onError.bind(this);
    }

    /**
     * 连接到SSE服务器
     */
    connect() {
        if (this.eventSource) {
            this.disconnect();
        }

        if (!this.token) {
            console.error('SSE Client: No authentication token provided');
            return;
        }

        const url = `${this.baseUrl}/api/v1/sse/events?client_id=${this.clientId}`;
        
        try {
            this.eventSource = new EventSource(url, {
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Accept': 'text/event-stream'
                }
            });

            this.eventSource.onopen = this.onOpen;
            this.eventSource.onmessage = this.onMessage;
            this.eventSource.onerror = this.onError;

            // 注册特定事件处理器
            this.registerEventHandlers();

            console.log('SSE Client: Connecting to', url);
        } catch (error) {
            console.error('SSE Client: Failed to create EventSource', error);
            this.scheduleReconnect();
        }
    }

    /**
     * 断开连接
     */
    disconnect() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        this.isConnected = false;
        this.reconnectAttempts = 0;
    }

    /**
     * 连接打开事件
     */
    onOpen(event) {
        console.log('SSE Client: Connected successfully');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.emit('connected', { clientId: this.clientId });
    }

    /**
     * 接收消息事件
     */
    onMessage(event) {
        try {
            const data = JSON.parse(event.data);
            console.log('SSE Client: Received event', data);
            
            // 触发通用消息事件
            this.emit('message', data);
            
            // 触发特定类型事件
            if (data.type) {
                this.emit(data.type, data);
            }
        } catch (error) {
            console.error('SSE Client: Failed to parse message', error, event.data);
        }
    }

    /**
     * 错误事件
     */
    onError(event) {
        console.error('SSE Client: Connection error', event);
        this.isConnected = false;
        this.emit('error', event);
        
        if (this.autoReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
        } else {
            console.error('SSE Client: Max reconnection attempts reached');
            this.emit('maxReconnectAttemptsReached');
        }
    }

    /**
     * 注册事件处理器
     */
    registerEventHandlers() {
        // 新邮件事件
        this.eventSource.addEventListener('new_email', (event) => {
            const data = JSON.parse(event.data);
            this.handleNewEmail(data);
        });

        // 邮件状态变更事件
        this.eventSource.addEventListener('email_read', (event) => {
            const data = JSON.parse(event.data);
            this.handleEmailStatusChange(data);
        });

        this.eventSource.addEventListener('email_starred', (event) => {
            const data = JSON.parse(event.data);
            this.handleEmailStatusChange(data);
        });

        // 同步事件
        this.eventSource.addEventListener('sync_started', (event) => {
            const data = JSON.parse(event.data);
            this.handleSyncEvent(data);
        });

        this.eventSource.addEventListener('sync_completed', (event) => {
            const data = JSON.parse(event.data);
            this.handleSyncEvent(data);
        });

        this.eventSource.addEventListener('sync_error', (event) => {
            const data = JSON.parse(event.data);
            this.handleSyncEvent(data);
        });

        // 通知事件
        this.eventSource.addEventListener('notification', (event) => {
            const data = JSON.parse(event.data);
            this.handleNotification(data);
        });

        // 心跳事件
        this.eventSource.addEventListener('heartbeat', (event) => {
            const data = JSON.parse(event.data);
            this.handleHeartbeat(data);
        });
    }

    /**
     * 处理新邮件事件
     */
    handleNewEmail(data) {
        console.log('New email received:', data);
        
        // 显示桌面通知
        if (this.isNotificationSupported()) {
            this.showDesktopNotification(
                '新邮件',
                `来自: ${data.data.from}\n主题: ${data.data.subject}`,
                'email'
            );
        }

        // 更新邮件列表
        this.emit('newEmail', data.data);
    }

    /**
     * 处理邮件状态变更事件
     */
    handleEmailStatusChange(data) {
        console.log('Email status changed:', data);
        this.emit('emailStatusChanged', data.data);
    }

    /**
     * 处理同步事件
     */
    handleSyncEvent(data) {
        console.log('Sync event:', data);
        this.emit('syncEvent', data.data);
    }

    /**
     * 处理通知事件
     */
    handleNotification(data) {
        console.log('Notification:', data);
        
        if (this.isNotificationSupported()) {
            this.showDesktopNotification(
                data.data.title,
                data.data.message,
                data.data.type
            );
        }

        this.emit('notification', data.data);
    }

    /**
     * 处理心跳事件
     */
    handleHeartbeat(data) {
        // 静默处理心跳，保持连接活跃
        this.emit('heartbeat', data.data);
    }

    /**
     * 安排重连
     */
    scheduleReconnect() {
        if (!this.autoReconnect) return;

        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectInterval * this.reconnectAttempts, 30000);
        
        console.log(`SSE Client: Scheduling reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
        
        setTimeout(() => {
            if (this.reconnectAttempts <= this.maxReconnectAttempts) {
                console.log(`SSE Client: Reconnection attempt ${this.reconnectAttempts}`);
                this.connect();
            }
        }, delay);
    }

    /**
     * 检查是否支持桌面通知
     */
    isNotificationSupported() {
        return 'Notification' in window && Notification.permission === 'granted';
    }

    /**
     * 请求桌面通知权限
     */
    async requestNotificationPermission() {
        if ('Notification' in window) {
            const permission = await Notification.requestPermission();
            return permission === 'granted';
        }
        return false;
    }

    /**
     * 显示桌面通知
     */
    showDesktopNotification(title, body, type = 'info') {
        if (!this.isNotificationSupported()) return;

        const options = {
            body: body,
            icon: '/static/images/email-icon.png',
            badge: '/static/images/badge-icon.png',
            tag: 'firemail-notification',
            requireInteraction: type === 'error'
        };

        const notification = new Notification(title, options);
        
        notification.onclick = () => {
            window.focus();
            notification.close();
        };

        // 自动关闭通知
        setTimeout(() => {
            notification.close();
        }, 5000);
    }

    /**
     * 事件监听
     */
    on(eventType, handler) {
        if (!this.eventHandlers.has(eventType)) {
            this.eventHandlers.set(eventType, []);
        }
        this.eventHandlers.get(eventType).push(handler);
    }

    /**
     * 移除事件监听
     */
    off(eventType, handler) {
        if (this.eventHandlers.has(eventType)) {
            const handlers = this.eventHandlers.get(eventType);
            const index = handlers.indexOf(handler);
            if (index > -1) {
                handlers.splice(index, 1);
            }
        }
    }

    /**
     * 触发事件
     */
    emit(eventType, data) {
        if (this.eventHandlers.has(eventType)) {
            this.eventHandlers.get(eventType).forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    console.error(`SSE Client: Error in event handler for ${eventType}`, error);
                }
            });
        }
    }

    /**
     * 生成客户端ID
     */
    generateClientId() {
        return 'client_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
    }

    /**
     * 获取连接状态
     */
    getConnectionState() {
        return {
            isConnected: this.isConnected,
            clientId: this.clientId,
            reconnectAttempts: this.reconnectAttempts
        };
    }

    /**
     * 更新认证token
     */
    updateToken(token) {
        this.token = token;
        if (this.isConnected) {
            // 重新连接以使用新token
            this.disconnect();
            this.connect();
        }
    }
}

// 导出类
if (typeof module !== 'undefined' && module.exports) {
    module.exports = FireMailSSEClient;
} else {
    window.FireMailSSEClient = FireMailSSEClient;
}
