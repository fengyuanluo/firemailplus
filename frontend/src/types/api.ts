/**
 * API相关类型定义
 * 消除any类型使用，提升类型安全
 */

// 基础API响应类型
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  code?: string | number;
  timestamp?: string;
}

// 分页响应类型
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

// 用户类型
export interface User {
  id: number;
  username: string;
  email: string;
  display_name?: string;
  avatar_url?: string;
  created_at: string;
  updated_at: string;
  is_active: boolean;
  preferences?: UserPreferences;
}

// 用户偏好设置
export interface UserPreferences {
  theme: 'light' | 'dark' | 'system';
  language: string;
  timezone: string;
  email_notifications: boolean;
  desktop_notifications: boolean;
  auto_sync_interval: number;
}

// 邮箱账户类型
export interface EmailAccount {
  id: number;
  user_id: number;
  email: string;
  display_name: string;
  provider: EmailProvider;
  is_active: boolean;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
  settings: EmailAccountSettings;
  sync_status: SyncStatus;
}

// 邮件提供商类型
export type EmailProvider = 'gmail' | 'outlook' | 'yahoo' | 'imap' | 'exchange';

// 邮箱账户设置
export interface EmailAccountSettings {
  sync_enabled: boolean;
  sync_interval: number;
  max_emails_per_sync: number;
  sync_folders: string[];
  signature?: string;
  auto_reply?: AutoReplySettings;
}

// 自动回复设置
export interface AutoReplySettings {
  enabled: boolean;
  subject: string;
  body: string;
  start_date?: string;
  end_date?: string;
}

// 同步状态
export interface SyncStatus {
  status: 'idle' | 'syncing' | 'error' | 'completed';
  last_sync_at?: string;
  next_sync_at?: string;
  error_message?: string;
  synced_count: number;
  total_count: number;
}

// 文件夹类型
export interface Folder {
  id: number;
  account_id: number;
  name: string;
  display_name: string;
  type: FolderType;
  parent_id?: number;
  children?: Folder[];
  unread_count: number;
  total_count: number;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

// 文件夹类型枚举
export type FolderType = 'inbox' | 'sent' | 'drafts' | 'trash' | 'spam' | 'archive' | 'custom';

// 邮件类型
export interface Email {
  id: number;
  account_id: number;
  folder_id: number;
  message_id: string;
  thread_id?: string;
  subject: string;
  from: EmailAddress;
  to: EmailAddress[];
  cc?: EmailAddress[];
  bcc?: EmailAddress[];
  reply_to?: EmailAddress[];
  date: string;
  body_text?: string;
  body_html?: string;
  attachments: Attachment[];
  is_read: boolean;
  is_starred: boolean;
  is_important: boolean;
  labels: Label[];
  flags: EmailFlag[];
  size: number;
  created_at: string;
  updated_at: string;
}

// 邮件地址类型
export interface EmailAddress {
  email: string;
  name?: string;
}

// 附件类型
export interface Attachment {
  id: number;
  email_id: number;
  filename: string;
  content_type: string;
  size: number;
  content_id?: string;
  disposition: string;
  storage_path?: string;
  is_downloaded: boolean;
  is_inline: boolean;
  download_url?: string;
  created_at: string;
  updated_at: string;
}

// 标签类型
export interface Label {
  id: number;
  account_id: number;
  name: string;
  color: string;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

// 邮件标记类型
export type EmailFlag = 'seen' | 'answered' | 'flagged' | 'deleted' | 'draft' | 'recent';

// 搜索过滤器类型
export interface SearchFilters {
  query?: string;
  from?: string;
  to?: string;
  subject?: string;
  has_attachment?: boolean;
  is_read?: boolean;
  is_starred?: boolean;
  date_from?: string;
  date_to?: string;
  folder_ids?: number[];
  label_ids?: number[];
  size_min?: number;
  size_max?: number;
}

// 邮件发送请求类型
export interface SendEmailRequest {
  account_id: number;
  to: EmailAddress[];
  cc?: EmailAddress[];
  bcc?: EmailAddress[];
  subject: string;
  body_text?: string;
  body_html?: string;
  attachments?: File[];
  reply_to_id?: number;
  forward_from_id?: number;
  is_draft?: boolean;
}

// 回复邮件请求类型
export interface ReplyEmailRequest {
  account_id: number;
  to: EmailAddress[];
  cc?: EmailAddress[];
  bcc?: EmailAddress[];
  subject: string;
  body_text?: string;
  body_html?: string;
  attachments?: File[];
}

// 转发邮件请求类型
export interface ForwardEmailRequest {
  account_id: number;
  to: EmailAddress[];
  cc?: EmailAddress[];
  bcc?: EmailAddress[];
  subject: string;
  body_text?: string;
  body_html?: string;
  attachments?: File[];
}

// 邮件操作类型
export type EmailAction =
  | 'mark_read'
  | 'mark_unread'
  | 'star'
  | 'unstar'
  | 'delete'
  | 'archive'
  | 'move'
  | 'label';

// 批量操作请求类型
export interface BulkEmailActionRequest {
  email_ids: number[];
  action: EmailAction;
  target_folder_id?: number;
  label_ids?: number[];
}

// 登录请求类型
export interface LoginRequest {
  username: string;
  password: string;
  remember_me?: boolean;
}

// 登录响应类型
export interface LoginResponse {
  user: User;
  token: string;
  expires_at: string;
  refresh_token?: string;
}

// 创建账户请求类型
export interface CreateAccountRequest {
  email: string;
  password: string;
  provider: EmailProvider;
  display_name?: string;
  imap_settings?: ImapSettings;
  smtp_settings?: SmtpSettings;
}

// IMAP设置类型
export interface ImapSettings {
  host: string;
  port: number;
  use_ssl: boolean;
  username: string;
  password: string;
}

// SMTP设置类型
export interface SmtpSettings {
  host: string;
  port: number;
  use_ssl: boolean;
  use_tls: boolean;
  username: string;
  password: string;
}

// 通知类型
export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  data?: Record<string, unknown>;
  is_read: boolean;
  created_at: string;
  expires_at?: string;
}

// 通知类型枚举
export type NotificationType =
  | 'info'
  | 'success'
  | 'warning'
  | 'error'
  | 'email_received'
  | 'sync_completed';

// 统计数据类型
export interface EmailStats {
  total_emails: number;
  unread_emails: number;
  starred_emails: number;
  today_emails: number;
  this_week_emails: number;
  this_month_emails: number;
  storage_used: number;
  storage_limit: number;
}

// 错误响应类型
export interface ErrorResponse {
  success: false;
  message: string;
  code?: string | number;
  details?: Record<string, unknown>;
  timestamp: string;
}

// 文件上传响应类型
export interface UploadResponse {
  id: string;
  filename: string;
  size: number;
  content_type: string;
  url: string;
  created_at: string;
}

// 导出/导入类型
export interface ExportRequest {
  account_ids?: number[];
  folder_ids?: number[];
  date_from?: string;
  date_to?: string;
  format: 'mbox' | 'eml' | 'json';
  include_attachments: boolean;
}

export interface ImportRequest {
  account_id: number;
  folder_id?: number;
  file: File;
  format: 'mbox' | 'eml' | 'json';
  merge_strategy: 'skip' | 'overwrite' | 'merge';
}

// SSE事件类型（从 @/types/sse 导入）
// 注意：SSE相关类型已移至 @/types/sse.ts 文件中

// 实时同步事件类型
export interface SyncEvent {
  account_id: number;
  event_type: 'sync_started' | 'sync_progress' | 'sync_completed' | 'sync_error';
  data: {
    progress?: number;
    total?: number;
    current?: number;
    error?: string;
    new_emails?: number;
  };
}

// 类型守卫函数
export function isApiResponse<T>(obj: unknown): obj is ApiResponse<T> {
  return typeof obj === 'object' && obj !== null && 'success' in obj;
}

export function isErrorResponse(obj: unknown): obj is ErrorResponse {
  return isApiResponse(obj) && obj.success === false;
}

export function isPaginatedResponse<T>(obj: unknown): obj is PaginatedResponse<T> {
  return typeof obj === 'object' && obj !== null && 'items' in obj && 'total' in obj;
}
