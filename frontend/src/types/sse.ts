/**
 * SSE (Server-Sent Events) 相关类型定义
 * 对应后端 SSE 事件结构
 */

// SSE 事件类型枚举
export type SSEEventType =
  | 'new_email'
  | 'email_read'
  | 'email_starred'
  | 'email_deleted'
  | 'sync_started'
  | 'sync_progress'
  | 'sync_completed'
  | 'sync_error'
  | 'account_connected'
  | 'account_disconnected'
  | 'account_error'
  | 'notification'
  | 'heartbeat';

// SSE 事件优先级
export type SSEEventPriority = 1 | 2 | 3 | 4; // 1=低, 2=普通, 3=高, 4=紧急

// 基础 SSE 事件结构
export interface SSEEvent<T = unknown> {
  id: string;
  type: SSEEventType;
  data: T;
  user_id?: number;
  account_id?: number;
  priority: SSEEventPriority;
  timestamp: string;
  retry?: number; // 重试间隔（毫秒）
}

// 新邮件事件数据
export interface NewEmailEventData {
  email_id: number;
  account_id: number;
  folder_id?: number;
  subject: string;
  from: string;
  date: string;
  is_read: boolean;
  has_attachment: boolean;
  preview?: string; // 邮件预览文本
}

// 邮件状态变更事件数据
export interface EmailStatusEventData {
  email_id: number;
  account_id: number;
  is_read?: boolean;
  is_starred?: boolean;
  is_important?: boolean;
  is_deleted?: boolean;
}

// 同步事件数据
export interface SyncEventData {
  account_id: number;
  account_name: string;
  status: string;
  progress?: number; // 0-100
  total_emails?: number;
  processed_emails?: number;
  error_message?: string;
}

// 账户事件数据
export interface AccountEventData {
  account_id: number;
  account_name: string;
  provider: string;
  status: 'connected' | 'disconnected' | 'error';
  error_message?: string;
}

// 通知事件数据
export interface NotificationEventData {
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
  duration?: number; // 显示时长（毫秒）
}

// 心跳事件数据
export interface HeartbeatEventData {
  server_time: string;
  client_id?: string;
}

// 具体的 SSE 事件类型
export type NewEmailEvent = SSEEvent<NewEmailEventData>;
export type EmailStatusEvent = SSEEvent<EmailStatusEventData>;
export type SyncEvent = SSEEvent<SyncEventData>;
export type AccountEvent = SSEEvent<AccountEventData>;
export type NotificationEvent = SSEEvent<NotificationEventData>;
export type HeartbeatEvent = SSEEvent<HeartbeatEventData>;

// SSE 事件联合类型
export type AnySSEEvent =
  | NewEmailEvent
  | EmailStatusEvent
  | SyncEvent
  | AccountEvent
  | NotificationEvent
  | HeartbeatEvent;

// SSE 客户端配置
export interface SSEClientConfig {
  baseUrl?: string;
  token: string;
  clientId?: string;
  autoReconnect?: boolean;
  reconnectInterval?: number; // 毫秒
  maxReconnectAttempts?: number;
  heartbeatTimeout?: number; // 毫秒
}

// SSE 客户端状态
export type SSEClientState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error';

// SSE 连接统计信息
export interface SSEConnectionStats {
  state: SSEClientState;
  connectedAt?: Date;
  lastHeartbeat?: Date;
  reconnectAttempts: number;
  totalEvents: number;
  eventsByType: Record<SSEEventType, number>;
}

// SSE 事件处理器类型
export type SSEEventHandler<T = unknown> = (event: SSEEvent<T>) => void;

// SSE 错误处理器类型
export type SSEErrorHandler = (error: Error) => void;

// SSE 状态变更处理器类型
export type SSEStateChangeHandler = (state: SSEClientState) => void;
