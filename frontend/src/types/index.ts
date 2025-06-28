/**
 * 统一类型导出
 * 提供项目中所有类型的统一入口
 */

// API相关类型
export type {
  ApiResponse,
  PaginatedResponse,
  User,
  UserPreferences,
  EmailAccount,
  EmailProvider,
  EmailAccountSettings,
  AutoReplySettings,
  SyncStatus,
  Folder,
  FolderType,
  Email,
  EmailAddress,
  Attachment,
  Label,
  EmailFlag,
  SearchFilters,
  SendEmailRequest,
  EmailAction,
  BulkEmailActionRequest,
  LoginRequest,
  LoginResponse,
  CreateAccountRequest,
  ImapSettings,
  SmtpSettings,
  Notification,
  NotificationType,
  EmailStats,
  ErrorResponse,
  UploadResponse,
  ExportRequest,
  ImportRequest,
  SyncEvent,
} from './api';

// 邮件相关类型
export type {
  EmailAccount as EmailAccountType,
  Email as EmailType,
  EmailAddress as EmailAddressType,
  Attachment as AttachmentType,
  Folder as EmailFolderType,
} from './email';

// SSE相关类型
export type {
  SSEEventType,
  SSEEventPriority,
  SSEEvent,
  NewEmailEventData,
  EmailStatusEventData,
  SyncEventData,
  AccountEventData,
  NotificationEventData,
  HeartbeatEventData,
  NewEmailEvent,
  EmailStatusEvent,
  SyncEvent as SSESyncEvent,
  AccountEvent,
  NotificationEvent,
  HeartbeatEvent,
  AnySSEEvent,
  SSEClientConfig,
  SSEClientState,
  SSEConnectionStats,
  SSEEventHandler,
  SSEErrorHandler,
  SSEStateChangeHandler,
} from './sse';

// 组件Props类型
export interface BaseComponentProps {
  className?: string;
  children?: React.ReactNode;
}

export interface LoadingProps extends BaseComponentProps {
  loading?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

export interface ErrorProps extends BaseComponentProps {
  error?: string | Error | null;
  onRetry?: () => void;
}

// 表单相关类型
export interface FormFieldProps {
  label?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
}

export interface SelectOption {
  value: string | number;
  label: string;
  disabled?: boolean;
}

// 布局相关类型
export interface LayoutProps extends BaseComponentProps {
  sidebar?: React.ReactNode;
  header?: React.ReactNode;
  footer?: React.ReactNode;
}

export interface ResponsiveProps {
  mobile?: boolean;
  tablet?: boolean;
  desktop?: boolean;
}

// 状态管理相关类型
export interface StoreState {
  loading: boolean;
  error: string | null;
  lastUpdated?: string;
}

export interface AsyncState<T> extends StoreState {
  data: T | null;
}

// Hook相关类型
export interface UseApiOptions {
  enabled?: boolean;
  refetchOnMount?: boolean;
  refetchOnWindowFocus?: boolean;
  staleTime?: number;
  cacheTime?: number;
}

export interface UseLoadingOptions {
  delay?: number;
  timeout?: number;
  showProgress?: boolean;
}

// 事件相关类型
export interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  metaKey?: boolean;
  action: string;
  description: string;
}

export interface ContextMenuAction {
  id: string;
  label: string;
  icon?: React.ReactNode;
  disabled?: boolean;
  separator?: boolean;
  onClick?: () => void;
}

// 主题相关类型
export type Theme = 'light' | 'dark' | 'system';

export interface ThemeConfig {
  theme: Theme;
  colors: {
    primary: string;
    secondary: string;
    background: string;
    foreground: string;
    muted: string;
    accent: string;
    destructive: string;
  };
}

// 路由相关类型
export interface RouteConfig {
  path: string;
  component: React.ComponentType;
  exact?: boolean;
  protected?: boolean;
  title?: string;
  description?: string;
}

// 搜索相关类型
export interface SearchResult<T> {
  items: T[];
  total: number;
  query: string;
  filters?: Record<string, any>;
  facets?: SearchFacet[];
}

export interface SearchFacet {
  name: string;
  values: SearchFacetValue[];
}

export interface SearchFacetValue {
  value: string;
  count: number;
  selected?: boolean;
}

// 分页相关类型
export interface PaginationConfig {
  page: number;
  pageSize: number;
  total: number;
  showSizeChanger?: boolean;
  showQuickJumper?: boolean;
  showTotal?: boolean;
}

// 排序相关类型
export interface SortConfig {
  field: string;
  order: 'asc' | 'desc';
}

// 过滤相关类型
export interface FilterConfig {
  field: string;
  operator: 'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'nin' | 'like' | 'regex';
  value: any;
}

// 导出导入相关类型
export interface ExportConfig {
  format: 'json' | 'csv' | 'xlsx' | 'pdf';
  fields?: string[];
  filters?: FilterConfig[];
  filename?: string;
}

export interface ImportConfig {
  format: 'json' | 'csv' | 'xlsx';
  mapping?: Record<string, string>;
  validation?: boolean;
  skipErrors?: boolean;
}

// 通知相关类型
export interface NotificationConfig {
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message?: string;
  duration?: number;
  closable?: boolean;
  actions?: NotificationAction[];
}

export interface NotificationAction {
  label: string;
  onClick: () => void;
  type?: 'primary' | 'secondary';
}

// 权限相关类型
export interface Permission {
  id: string;
  name: string;
  description?: string;
  resource: string;
  action: string;
}

export interface Role {
  id: string;
  name: string;
  description?: string;
  permissions: Permission[];
}

// 配置相关类型
export interface AppConfig {
  name: string;
  version: string;
  apiUrl: string;
  sseUrl?: string; // SSE连接URL（可选，默认使用apiUrl）
  theme: ThemeConfig;
  features: Record<string, boolean>;
  limits: Record<string, number>;
}

// 工具类型
export type Optional<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;
export type RequiredFields<T, K extends keyof T> = T & Required<Pick<T, K>>;
export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};
export type DeepRequired<T> = {
  [P in keyof T]-?: T[P] extends object ? DeepRequired<T[P]> : T[P];
};

// 联合类型工具
export type UnionToIntersection<U> = (U extends any ? (k: U) => void : never) extends (
  k: infer I
) => void
  ? I
  : never;
export type LastOf<T> =
  UnionToIntersection<T extends any ? () => T : never> extends () => infer R ? R : never;

// 函数类型工具
export type AsyncFunction<T extends any[] = any[], R = any> = (...args: T) => Promise<R>;
export type EventHandler<T = any> = (event: T) => void;
export type Callback<T = void> = () => T;

// 状态机相关类型
export interface StateMachine<S extends string, E extends string> {
  state: S;
  transitions: Record<S, Partial<Record<E, S>>>;
  onTransition?: (from: S, to: S, event: E) => void;
}

// 缓存相关类型
export interface CacheConfig {
  ttl?: number;
  maxSize?: number;
  strategy?: 'lru' | 'fifo' | 'lfu';
}

export interface CacheEntry<T> {
  value: T;
  timestamp: number;
  ttl?: number;
  hits?: number;
}

// 验证相关类型
export interface ValidationRule<T = any> {
  required?: boolean;
  min?: number;
  max?: number;
  pattern?: RegExp;
  custom?: (value: T) => boolean | string;
  message?: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: Record<string, string>;
}

// 国际化相关类型
export interface I18nConfig {
  locale: string;
  fallback: string;
  messages: Record<string, Record<string, string>>;
}

// 性能监控相关类型
export interface PerformanceMetric {
  name: string;
  value: number;
  unit: string;
  timestamp: number;
  tags?: Record<string, string>;
}

export interface PerformanceReport {
  metrics: PerformanceMetric[];
  summary: {
    totalTime: number;
    renderCount: number;
    memoryUsage?: number;
  };
}
