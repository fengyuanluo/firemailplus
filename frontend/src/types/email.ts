/**
 * 邮件相关的TypeScript接口定义
 */

// 邮件地址接口
export interface EmailAddress {
  name: string;
  address: string;
}

// 邮箱账户接口
export interface EmailAccount {
  id: number;
  user_id: number;
  name: string;
  email: string;
  provider: string;
  auth_method: string;
  username?: string; // 用户名（用于自定义邮箱）

  // IMAP配置
  imap_host: string;
  imap_port: number;
  imap_security: string;

  // SMTP配置
  smtp_host: string;
  smtp_port: number;
  smtp_security: string;

  // 状态信息
  is_active: boolean;
  last_sync_at?: string;
  sync_status: string;
  error_message?: string;

  // 统计信息
  total_emails: number;
  unread_emails: number;

  // 时间戳
  created_at: string;
  updated_at: string;

  // 关联关系
  folders?: Folder[];
}

// 文件夹接口
export interface Folder {
  id: number;
  account_id: number;
  name: string;
  display_name: string;
  type: string; // inbox, sent, drafts, trash, spam, custom
  parent_id?: number;
  path: string;
  delimiter: string;

  // 文件夹属性
  is_selectable: boolean;
  is_subscribed: boolean;

  // 统计信息
  total_emails: number;
  unread_emails: number;

  // 同步信息
  uid_validity: number;
  uid_next: number;

  // 时间戳
  created_at: string;
  updated_at: string;

  // 关联关系
  parent?: Folder;
  children?: Folder[];
}

// 附件接口
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

// 邮件接口
export interface Email {
  id: number;
  account_id: number;
  folder_id?: number;
  message_id: string;
  uid: number;

  // 邮件头信息
  subject: string;
  from: string; // JSON字符串，需要解析为EmailAddress
  to: string; // JSON字符串，需要解析为EmailAddress[]
  cc: string; // JSON字符串，需要解析为EmailAddress[]
  bcc: string; // JSON字符串，需要解析为EmailAddress[]
  reply_to: string;
  date: string;

  // 邮件内容
  text_body: string;
  html_body: string;

  // 邮件状态
  is_read: boolean;
  is_starred: boolean;
  is_important: boolean;
  is_deleted: boolean;
  is_draft: boolean;
  is_sent: boolean;

  // 邮件大小和附件信息
  size: number;
  has_attachment: boolean;

  // 邮件标签和分类
  labels: string; // JSON字符串
  priority: string; // low, normal, high

  // 同步信息
  synced_at?: string;

  // 时间戳
  created_at: string;
  updated_at: string;

  // 关联关系
  account?: EmailAccount;
  folder?: Folder;
  attachments?: Attachment[];

  // 前端扩展字段
  preview?: string; // 邮件正文预览
  parsed_from?: EmailAddress; // 解析后的发件人
  parsed_to?: EmailAddress[]; // 解析后的收件人
  parsed_cc?: EmailAddress[]; // 解析后的抄送人
  parsed_bcc?: EmailAddress[]; // 解析后的密送人
}

// 邮件列表请求参数
export interface GetEmailsRequest {
  account_id?: number;
  folder_id?: number;
  is_read?: boolean;
  is_starred?: boolean;
  is_important?: boolean;
  search?: string;
  page?: number;
  page_size?: number;
  sort_by?: string; // date, subject, from, size
  sort_order?: string; // asc, desc
}

// 邮件列表响应
export interface GetEmailsResponse {
  emails: Email[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// 文件夹类型常量
export const FolderType = {
  INBOX: 'inbox',
  SENT: 'sent',
  DRAFTS: 'drafts',
  TRASH: 'trash',
  SPAM: 'spam',
  CUSTOM: 'custom',
} as const;

// 邮件优先级常量
export const EmailPriority = {
  LOW: 'low',
  NORMAL: 'normal',
  HIGH: 'high',
} as const;

// 同步状态常量
export const SyncStatus = {
  PENDING: 'pending',
  SYNCING: 'syncing',
  SUCCESS: 'success',
  ERROR: 'error',
} as const;

// 邮件操作类型
export type EmailOperation =
  | 'read'
  | 'unread'
  | 'star'
  | 'unstar'
  | 'important'
  | 'unimportant'
  | 'delete'
  | 'move'
  | 'archive';

// 批量邮件操作请求
export interface BatchEmailOperationRequest {
  email_ids: number[];
  operation: EmailOperation;
  target_folder_id?: number; // 用于move操作
}

// 邮件搜索请求
export interface SearchEmailsRequest extends GetEmailsRequest {
  q?: string; // 通用搜索关键词
  subject?: string; // 主题搜索
  from?: string; // 发件人搜索
  to?: string; // 收件人搜索
  body?: string; // 正文搜索
  since?: string; // 开始时间 (RFC3339格式)
  before?: string; // 结束时间 (RFC3339格式)
  has_attachment?: boolean; // 是否有附件
}

// 邮件统计信息
export interface EmailStats {
  total_emails: number;
  unread_emails: number;
  starred_emails: number;
  important_emails: number;
  emails_with_attachments: number;
  total_size: number;
  by_account: Array<{
    account_id: number;
    account_name: string;
    total: number;
    unread: number;
  }>;
  by_folder: Array<{
    folder_id: number;
    folder_name: string;
    total: number;
    unread: number;
  }>;
}

// 工具函数：解析邮件地址JSON字符串
export function parseEmailAddresses(addressesJson: string): EmailAddress[] {
  if (!addressesJson) return [];
  try {
    const parsed = JSON.parse(addressesJson);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    // 如果JSON解析失败，检查是否是普通邮箱地址字符串（可能用逗号分隔）
    if (addressesJson.includes('@')) {
      return addressesJson
        .split(',')
        .map((addr) => ({
          name: '',
          address: addr.trim(),
        }))
        .filter((addr) => addr.address.includes('@'));
    }
    return [];
  }
}

// 工具函数：解析单个邮件地址JSON字符串
export function parseEmailAddress(addressJson: string): EmailAddress | null {
  if (!addressJson) return null;
  try {
    return JSON.parse(addressJson);
  } catch {
    // 如果JSON解析失败，检查是否是普通邮箱地址字符串
    if (addressJson.includes('@')) {
      return {
        name: '',
        address: addressJson.trim(),
      };
    }
    return null;
  }
}

// 工具函数：格式化邮件地址显示
export function formatEmailAddress(address: EmailAddress): string {
  if (address.name) {
    return `${address.name} <${address.address}>`;
  }
  return address.address;
}

// 工具函数：获取邮件预览文本
export function getEmailPreview(email: Email, maxLength: number = 100): string {
  const content = email.text_body || email.html_body || '';
  // 移除HTML标签
  const textContent = content.replace(/<[^>]*>/g, '');
  // 移除多余空白字符
  const cleanText = textContent.replace(/\s+/g, ' ').trim();
  // 截断并添加省略号
  if (cleanText.length > maxLength) {
    return cleanText.substring(0, maxLength) + '...';
  }
  return cleanText;
}

// 工具函数：检查文件夹是否为系统文件夹
export function isSystemFolder(folder: Folder): boolean {
  const systemTypes = [
    FolderType.INBOX,
    FolderType.SENT,
    FolderType.DRAFTS,
    FolderType.TRASH,
    FolderType.SPAM,
  ];
  return systemTypes.includes(folder.type as any);
}

// 工具函数：格式化文件大小
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}
