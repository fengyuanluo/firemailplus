/**
 * SSE (Server-Sent Events) 相关类型定义
 * 对应后端 SSE 事件结构
 */

export const ALL_SSE_EVENT_TYPES = [
  'new_email',
  'email_read',
  'email_unread',
  'email_starred',
  'email_unstarred',
  'email_important',
  'email_unimportant',
  'email_deleted',
  'email_moved',
  'folder_read_state_changed',
  'account_read_state_changed',
  'sync_started',
  'sync_progress',
  'sync_completed',
  'sync_error',
  'account_connected',
  'account_disconnected',
  'account_error',
  'group_created',
  'group_updated',
  'group_deleted',
  'group_reordered',
  'group_default_changed',
  'account_group_changed',
  'notification',
  'heartbeat',
] as const;

// SSE 事件类型枚举
export type SSEEventType = (typeof ALL_SSE_EVENT_TYPES)[number];

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
  folder_id?: number;
  is_read?: boolean;
  is_starred?: boolean;
  is_important?: boolean;
  is_deleted?: boolean;
  unread_delta?: number;
}

// 邮件移动事件数据
export interface EmailMovedEventData {
  email_id: number;
  account_id: number;
  source_folder_id?: number;
  target_folder_id: number;
  is_read: boolean;
}

// 文件夹批量读状态变更事件数据
export interface FolderReadStateEventData {
  account_id: number;
  folder_id: number;
  affected_count: number;
}

// 账户批量读状态变更事件数据
export interface AccountReadStateEventData {
  account_id: number;
  affected_count: number;
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

// 邮箱分组事件数据
export interface GroupEventData {
  group_id?: number;
  name?: string;
  sort_order?: number;
  is_default?: boolean;
  system_key?: string | null;
  group_ids?: number[];
  previous_default_group_id?: number;
}

// 账户分组事件数据
export interface AccountGroupEventData {
  account_id: number;
  account_name: string;
  email: string;
  group_id?: number;
  previous_group_id?: number;
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
export type EmailMovedEvent = SSEEvent<EmailMovedEventData>;
export type FolderReadStateEvent = SSEEvent<FolderReadStateEventData>;
export type AccountReadStateEvent = SSEEvent<AccountReadStateEventData>;
export type SyncEvent = SSEEvent<SyncEventData>;
export type AccountEvent = SSEEvent<AccountEventData>;
export type GroupEvent = SSEEvent<GroupEventData>;
export type AccountGroupEvent = SSEEvent<AccountGroupEventData>;
export type NotificationEvent = SSEEvent<NotificationEventData>;
export type HeartbeatEvent = SSEEvent<HeartbeatEventData>;

// SSE 事件联合类型
export type AnySSEEvent =
  | NewEmailEvent
  | EmailStatusEvent
  | EmailMovedEvent
  | FolderReadStateEvent
  | AccountReadStateEvent
  | SyncEvent
  | AccountEvent
  | GroupEvent
  | AccountGroupEvent
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
