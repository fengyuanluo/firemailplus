/**
 * FireMail SSE 客户端
 * TypeScript 版本的 Server-Sent Events 客户端
 */

import type {
  SSEClientConfig,
  SSEClientState,
  SSEConnectionStats,
  SSEEventType,
  AnySSEEvent,
  SSEEventHandler,
  SSEErrorHandler,
  SSEStateChangeHandler,
} from '@/types/sse';

export class FireMailSSEClient {
  private config: Required<SSEClientConfig>;
  private eventSource: EventSource | null = null;
  private state: SSEClientState = 'disconnected';
  private reconnectAttempts = 0;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private stats: SSEConnectionStats;

  // 事件处理器映射
  private eventHandlers = new Map<SSEEventType, Set<SSEEventHandler>>();
  private errorHandlers = new Set<SSEErrorHandler>();
  private stateChangeHandlers = new Set<SSEStateChangeHandler>();

  constructor(config: SSEClientConfig) {
    this.config = {
      baseUrl: config.baseUrl || '',
      token: config.token,
      clientId: config.clientId || this.generateClientId(),
      autoReconnect: config.autoReconnect !== false,
      reconnectInterval: config.reconnectInterval || 5000,
      maxReconnectAttempts: config.maxReconnectAttempts || 10,
      heartbeatTimeout: config.heartbeatTimeout || 60000,
    };

    this.stats = {
      state: 'disconnected',
      reconnectAttempts: 0,
      totalEvents: 0,
      eventsByType: {} as Record<SSEEventType, number>,
    };

    // 初始化事件计数器
    this.initializeEventCounters();
  }

  /**
   * 连接到 SSE 服务器
   */
  public connect(): void {
    if (this.eventSource) {
      this.disconnect();
    }

    if (!this.config.token) {
      this.handleError(new Error('认证令牌未提供'));
      return;
    }

    this.setState('connecting');

    const url = `${this.config.baseUrl}/sse/events?client_id=${this.config.clientId}&token=${encodeURIComponent(this.config.token)}`;

    try {
      this.eventSource = new EventSource(url);
      this.setupEventListeners();
      console.log('🔗 [SSEClient] 正在连接到:', url);
    } catch (error) {
      this.handleError(error as Error);
      this.scheduleReconnect();
    }
  }

  /**
   * 断开连接
   */
  public disconnect(): void {
    this.clearTimers();

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    this.setState('disconnected');
    this.reconnectAttempts = 0;
    console.log('🔌 [SSEClient] 连接已断开');
  }

  /**
   * 更新认证令牌
   */
  public updateToken(token: string): void {
    this.config.token = token;
    if (this.state === 'connected') {
      // 重新连接以使用新令牌
      this.connect();
    }
  }

  /**
   * 添加事件处理器
   */
  public on<T = unknown>(eventType: SSEEventType, handler: SSEEventHandler<T>): void {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, new Set());
    }
    this.eventHandlers.get(eventType)!.add(handler as SSEEventHandler);
  }

  /**
   * 移除事件处理器
   */
  public off<T = unknown>(eventType: SSEEventType, handler: SSEEventHandler<T>): void {
    const handlers = this.eventHandlers.get(eventType);
    if (handlers) {
      handlers.delete(handler as SSEEventHandler);
    }
  }

  /**
   * 添加错误处理器
   */
  public onError(handler: SSEErrorHandler): void {
    this.errorHandlers.add(handler);
  }

  /**
   * 添加状态变更处理器
   */
  public onStateChange(handler: SSEStateChangeHandler): void {
    this.stateChangeHandlers.add(handler);
  }

  /**
   * 获取当前状态
   */
  public getState(): SSEClientState {
    return this.state;
  }

  /**
   * 获取连接统计信息
   */
  public getStats(): SSEConnectionStats {
    return { ...this.stats };
  }

  /**
   * 检查是否已连接
   */
  public isConnected(): boolean {
    return this.state === 'connected';
  }

  // 私有方法

  private setupEventListeners(): void {
    if (!this.eventSource) return;

    this.eventSource.onopen = this.handleOpen.bind(this);
    this.eventSource.onmessage = this.handleMessage.bind(this);
    this.eventSource.onerror = this.handleEventSourceError.bind(this);

    // 注册特定事件类型的监听器
    this.registerSpecificEventListeners();
  }

  private registerSpecificEventListeners(): void {
    if (!this.eventSource) return;

    const eventTypes: SSEEventType[] = [
      'new_email',
      'email_read',
      'email_starred',
      'email_deleted',
      'sync_started',
      'sync_progress',
      'sync_completed',
      'sync_error',
      'account_connected',
      'account_disconnected',
      'account_error',
      'notification',
      'heartbeat',
    ];

    eventTypes.forEach((eventType) => {
      this.eventSource!.addEventListener(eventType, (event) => {
        this.handleTypedEvent(eventType, event as MessageEvent);
      });
    });
  }

  private handleOpen(): void {
    console.log('✅ [SSEClient] 连接已建立');
    this.setState('connected');
    this.reconnectAttempts = 0;
    this.stats.connectedAt = new Date();
    this.startHeartbeatMonitor();
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data) as AnySSEEvent;
      this.processEvent(data);
    } catch (error) {
      console.error('❌ [SSEClient] 解析消息失败:', error, event.data);
    }
  }

  private handleTypedEvent(eventType: SSEEventType, event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data) as AnySSEEvent;
      this.processEvent(data);
    } catch (error) {
      console.error(`❌ [SSEClient] 解析 ${eventType} 事件失败:`, error, event.data);
    }
  }

  private handleEventSourceError(event: Event): void {
    const readyState = this.eventSource?.readyState ?? 'unknown';
    const url = `${this.config.baseUrl}/sse/events?client_id=${this.config.clientId}&token=${encodeURIComponent(this.config.token)}`;

    let errorMessage = 'EventSource 连接错误';
    switch (readyState) {
      case EventSource.CONNECTING:
        errorMessage = '正在尝试连接到 SSE 服务器';
        break;
      case EventSource.OPEN:
        errorMessage = '连接已建立但发生错误';
        break;
      case EventSource.CLOSED:
        errorMessage = '连接已关闭或无法建立';
        break;
      default:
        errorMessage = `未知连接状态: ${readyState}`;
    }

    console.error('❌ [SSEClient] EventSource 错误:', errorMessage, {
      readyState,
      url,
      eventType: event?.type || 'unknown',
    });

    if (readyState === EventSource.CLOSED) {
      this.setState('error');
      this.scheduleReconnect();
    }
  }

  private processEvent(event: AnySSEEvent): void {
    console.log(`📨 [SSEClient] 收到事件:`, event.type, event);

    // 更新统计信息
    this.stats.totalEvents++;
    this.stats.eventsByType[event.type] = (this.stats.eventsByType[event.type] || 0) + 1;

    // 处理心跳事件
    if (event.type === 'heartbeat') {
      this.stats.lastHeartbeat = new Date();
      this.resetHeartbeatMonitor();
    }

    // 触发事件处理器
    const handlers = this.eventHandlers.get(event.type);
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler(event);
        } catch (error) {
          console.error(`❌ [SSEClient] 事件处理器错误 (${event.type}):`, error);
        }
      });
    }
  }

  private setState(newState: SSEClientState): void {
    if (this.state !== newState) {
      const oldState = this.state;
      this.state = newState;
      this.stats.state = newState;

      console.log(`🔄 [SSEClient] 状态变更: ${oldState} -> ${newState}`);

      // 触发状态变更处理器
      this.stateChangeHandlers.forEach((handler) => {
        try {
          handler(newState);
        } catch (error) {
          console.error('❌ [SSEClient] 状态变更处理器错误:', error);
        }
      });
    }
  }

  private handleError(error: Error): void {
    console.error('❌ [SSEClient] 错误:', error);
    this.setState('error');

    // 触发错误处理器
    this.errorHandlers.forEach((handler) => {
      try {
        handler(error);
      } catch (handlerError) {
        console.error('❌ [SSEClient] 错误处理器异常:', handlerError);
      }
    });
  }

  private scheduleReconnect(): void {
    if (!this.config.autoReconnect || this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      console.log('🚫 [SSEClient] 已达到最大重连次数或禁用自动重连');
      return;
    }

    this.reconnectAttempts++;
    this.stats.reconnectAttempts = this.reconnectAttempts;

    const delay = Math.min(
      this.config.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1),
      30000 // 最大延迟 30 秒
    );

    console.log(
      `🔄 [SSEClient] 计划重连 (${this.reconnectAttempts}/${this.config.maxReconnectAttempts}) 延迟: ${delay}ms`
    );

    this.setState('reconnecting');
    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  private startHeartbeatMonitor(): void {
    this.resetHeartbeatMonitor();
  }

  private resetHeartbeatMonitor(): void {
    if (this.heartbeatTimer) {
      clearTimeout(this.heartbeatTimer);
    }

    this.heartbeatTimer = setTimeout(() => {
      console.warn('⚠️ [SSEClient] 心跳超时，可能连接已断开');
      this.handleError(new Error('心跳超时'));
      this.scheduleReconnect();
    }, this.config.heartbeatTimeout);
  }

  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.heartbeatTimer) {
      clearTimeout(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private generateClientId(): string {
    return `client_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private initializeEventCounters(): void {
    const eventTypes: SSEEventType[] = [
      'new_email',
      'email_read',
      'email_starred',
      'email_deleted',
      'sync_started',
      'sync_progress',
      'sync_completed',
      'sync_error',
      'account_connected',
      'account_disconnected',
      'account_error',
      'notification',
      'heartbeat',
    ];

    eventTypes.forEach((type) => {
      this.stats.eventsByType[type] = 0;
    });
  }
}
