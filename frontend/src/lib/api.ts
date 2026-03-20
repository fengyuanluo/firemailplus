/**
 * API 配置和基础请求函数
 */

import type {
  EmailAccount,
  Email,
  EmailAddress,
  EmailStats,
  Folder,
  EmailGroup,
} from '@/types/email';

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1';

// API 响应类型
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

type ApiError = Error & {
  status?: number;
  data?: unknown;
};

// 认证相关类型
export interface LoginRequest {
  username: string;
  password: string;
}

export interface User {
  id: number;
  username: string;
  email?: string;
  display_name?: string;
  role: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
  login_count: number;
}

export interface LoginResponse {
  token: string;
  expires_at: string;
  user: User;
}

// 邮箱账户创建请求类型

export interface CreateAccountRequest {
  name: string;
  email: string;
  provider: string;
  auth_method: string;
  username?: string;
  password?: string;
  imap_host?: string;
  imap_port?: number;
  imap_security?: string;
  smtp_host?: string;
  smtp_port?: number;
  smtp_security?: string;
  group_id?: number;
}

// 基础请求函数
class ApiClient {
  private getAuthToken(): string | null {
    if (typeof window !== 'undefined') {
      // 从Zustand persist存储中获取token
      const authStorage = localStorage.getItem('auth-storage');
      if (authStorage) {
        try {
          const parsed = JSON.parse(authStorage);
          return parsed.state?.token || null;
        } catch (error) {
          console.error('Failed to parse auth storage:', error);
          return null;
        }
      }
    }
    return null;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    const token = this.getAuthToken();
    const url = `${API_BASE_URL}${endpoint}`;

    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);

      // 尝试解析JSON响应
      let data: ApiResponse<T>;
      try {
        data = (await response.json()) as ApiResponse<T>;
      } catch {
        throw new Error(`服务器响应格式错误: ${response.status}`);
      }

      if (!response.ok) {
        // 根据状态码提供更友好的错误消息
        let errorMessage = data.message || data.error || '请求失败';

        switch (response.status) {
          case 401:
            errorMessage = data.message || '登录已过期，请重新登录';
            // 401 错误表示认证失败，需要清除认证状态
            if (typeof window !== 'undefined') {
              // 清除认证存储
              localStorage.removeItem('auth-storage');
              // 延迟重定向，让错误处理完成
              setTimeout(() => {
                window.location.href = '/login';
              }, 100);
            }
            break;
          case 403:
            errorMessage = data.message || '账户已被禁用';
            break;
          case 404:
            errorMessage = data.message || '请求的资源不存在';
            break;
          case 500:
            errorMessage = data.message || '服务器内部错误';
            break;
          default:
            errorMessage = data.message || `请求失败 (${response.status})`;
        }

        const apiError = new Error(errorMessage) as ApiError;
        apiError.status = response.status;
        apiError.data = data;
        throw apiError;
      }

      return data;
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // 认证相关 API
  async login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }

  async logout(): Promise<ApiResponse> {
    return this.request('/auth/logout', {
      method: 'POST',
    });
  }

  async getCurrentUser(): Promise<ApiResponse<LoginResponse['user']>> {
    return this.request('/auth/me');
  }

  // 邮箱账户相关 API
  async getEmailAccounts(): Promise<ApiResponse<EmailAccount[]>> {
    return this.request('/accounts');
  }

  async getEmailAccount(id: number): Promise<ApiResponse<EmailAccount>> {
    return this.request(`/accounts/${id}`);
  }

  async createEmailAccount(account: CreateAccountRequest): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>('/accounts', {
      method: 'POST',
      body: JSON.stringify(account),
    });
  }

  async updateEmailAccount(
    id: number,
    data: {
      name?: string;
      password?: string;
      imap_host?: string;
      imap_port?: number;
      imap_security?: string;
      smtp_host?: string;
      smtp_port?: number;
      smtp_security?: string;
      is_active?: boolean;
      group_id?: number | null;
    }
  ): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>(`/accounts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async testEmailAccount(id: number): Promise<ApiResponse> {
    return this.request(`/accounts/${id}/test`, {
      method: 'POST',
    });
  }

  async deleteEmailAccount(id: number): Promise<ApiResponse> {
    return this.request(`/accounts/${id}`, {
      method: 'DELETE',
    });
  }

  async batchDeleteEmailAccounts(accountIds: number[]): Promise<ApiResponse> {
    return this.request('/accounts/batch/delete', {
      method: 'POST',
      body: JSON.stringify({ account_ids: accountIds }),
    });
  }

  async batchSyncEmailAccounts(accountIds: number[]): Promise<ApiResponse> {
    return this.request('/accounts/batch/sync', {
      method: 'POST',
      body: JSON.stringify({ account_ids: accountIds }),
    });
  }

  async markAccountAsRead(accountId: number): Promise<ApiResponse> {
    return this.request(`/accounts/${accountId}/mark-read`, {
      method: 'PUT',
    });
  }

  async batchMarkAccountsAsRead(accountIds: number[]): Promise<ApiResponse> {
    return this.request('/accounts/batch/mark-read', {
      method: 'POST',
      body: JSON.stringify({ account_ids: accountIds }),
    });
  }

  // 邮箱分组相关 API
  async getEmailGroups(): Promise<ApiResponse<EmailGroup[]>> {
    return this.request('/groups');
  }

  async createEmailGroup(payload: { name: string }): Promise<ApiResponse<EmailGroup>> {
    return this.request<EmailGroup>('/groups', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async updateEmailGroup(id: number, payload: { name?: string }): Promise<ApiResponse<EmailGroup>> {
    return this.request<EmailGroup>(`/groups/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  async deleteEmailGroup(id: number): Promise<ApiResponse> {
    return this.request(`/groups/${id}`, {
      method: 'DELETE',
    });
  }

  async setDefaultEmailGroup(id: number): Promise<ApiResponse<EmailGroup>> {
    return this.request<EmailGroup>(`/groups/${id}/default`, {
      method: 'PUT',
    });
  }

  async reorderEmailGroups(groupIds: number[]): Promise<ApiResponse<EmailGroup[]>> {
    return this.request<EmailGroup[]>('/groups/reorder', {
      method: 'PUT',
      body: JSON.stringify({ group_ids: groupIds }),
    });
  }

  // OAuth2 相关 API
  async getGmailOAuthUrl(
    callbackUrl?: string
  ): Promise<ApiResponse<{ auth_url: string; state: string }>> {
    const url = callbackUrl
      ? `/oauth/gmail?callback_url=${encodeURIComponent(callbackUrl)}`
      : '/oauth/gmail';
    return this.request(url);
  }

  async getOutlookOAuthUrl(
    callbackUrl?: string
  ): Promise<ApiResponse<{ auth_url: string; state: string }>> {
    const url = callbackUrl
      ? `/oauth/outlook?callback_url=${encodeURIComponent(callbackUrl)}`
      : '/oauth/outlook';
    return this.request(url);
  }

  // 通过后端API处理OAuth2回调（后端会调用外部OAuth服务器）
  async handleOAuth2Callback(
    provider: string,
    code: string,
    state: string
  ): Promise<
    ApiResponse<{
      access_token: string;
      refresh_token?: string;
      token_type: string;
      expires_in: number;
      scope?: string;
    }>
  > {
    // 调用后端的OAuth回调处理端点
    return this.request(
      `/oauth/${provider}/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
      {
        method: 'GET',
      }
    );
  }

  async createOAuth2Account(account: {
    name: string;
    email: string;
    provider: string;
    access_token: string;
    refresh_token: string; // 必需，用于token验证和刷新
    expires_at: number;
    scope?: string;
    client_id: string; // 必需，用于token刷新
    group_id?: number;
  }): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>('/oauth/create-account', {
      method: 'POST',
      body: JSON.stringify(account),
    });
  }

  async createManualOAuth2Account(account: {
    name: string;
    email: string;
    provider: string;
    client_id: string;
    client_secret?: string;
    refresh_token: string;
    scope?: string;
    auth_url?: string;
    token_url?: string;
    group_id?: number;
  }): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>('/oauth/manual-config', {
      method: 'POST',
      body: JSON.stringify(account),
    });
  }

  async createCustomEmailAccount(data: {
    name: string;
    email: string;
    auth_method: string;
    username: string;
    password: string;
    imap_host?: string;
    imap_port?: number;
    imap_security?: string;
    smtp_host?: string;
    smtp_port?: number;
    smtp_security?: string;
    group_id?: number;
  }): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>('/accounts/custom', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // 邮件相关 API
  async getEmails(params?: {
    account_id?: number;
    folder_id?: number;
    page?: number;
    page_size?: number;
    search?: string;
    is_read?: boolean;
    is_starred?: boolean;
    is_important?: boolean;
    sort_by?: string;
    sort_order?: string;
  }): Promise<ApiResponse<{ emails: Email[]; total: number; page: number; page_size: number }>> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.append(key, value.toString());
        }
      });
    }

    const query = searchParams.toString();
    return this.request(`/emails${query ? `?${query}` : ''}`);
  }

  async searchEmails(params: {
    q?: string; // 全文搜索关键词
    subject?: string; // 主题搜索
    from?: string; // 发件人搜索
    to?: string; // 收件人搜索
    body?: string; // 正文搜索
    since?: string; // 开始时间 (RFC3339格式)
    before?: string; // 结束时间 (RFC3339格式)
    has_attachment?: boolean; // 是否有附件
    is_read?: boolean; // 是否已读
    is_starred?: boolean; // 是否加星
    account_id?: number; // 账户ID筛选
    folder_id?: number; // 文件夹ID筛选
    page?: number; // 页码
    page_size?: number; // 每页大小
  }): Promise<
    ApiResponse<{
      emails: Email[];
      total: number;
      page: number;
      page_size: number;
      total_pages?: number;
    }>
  > {
    console.log('🌐 [ApiClient] searchEmails() 被调用:', params);

    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, value.toString());
      }
    });

    const url = `/emails/search?${searchParams.toString()}`;
    console.log('🌐 [ApiClient] 请求URL:', url);

    try {
      const result = await this.request<{
        emails: Email[];
        total: number;
        page: number;
        page_size: number;
        total_pages?: number;
      }>(url);
      console.log('🌐 [ApiClient] searchEmails 响应:', {
        success: result.success,
        dataExists: !!result.data,
        emailCount: result.data?.emails?.length || 0,
        total: result.data?.total || 0,
        message: result.message,
      });
      return result;
    } catch (error) {
      console.error('🌐 [ApiClient] searchEmails 错误:', error);
      throw error;
    }
  }

  async getFolders(accountId?: number): Promise<ApiResponse<Folder[]>> {
    const params = accountId ? `?account_id=${accountId}` : '';
    return this.request(`/folders${params}`);
  }

  async getFolder(folderId: number): Promise<ApiResponse<Folder>> {
    return this.request(`/folders/${folderId}`);
  }

  async getEmailStats(): Promise<ApiResponse<EmailStats>> {
    return this.request('/emails/stats');
  }

  async batchEmailOperation(data: {
    email_ids: number[];
    operation: string;
    target_folder_id?: number;
  }): Promise<ApiResponse> {
    return this.request('/emails/batch', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async markAllAsRead(folderId: number): Promise<ApiResponse> {
    return this.request(`/folders/${folderId}/mark-all-read`, {
      method: 'PUT',
    });
  }

  async moveEmail(emailId: number, targetFolderId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}/move`, {
      method: 'PUT',
      body: JSON.stringify({ target_folder_id: targetFolderId }),
    });
  }

  async createFolder(data: {
    account_id: number;
    name: string;
    display_name?: string;
    parent_id?: number;
  }): Promise<ApiResponse<Folder>> {
    return this.request('/folders', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateFolder(
    folderId: number,
    data: {
      name?: string;
      display_name?: string;
    }
  ): Promise<ApiResponse<Folder>> {
    return this.request(`/folders/${folderId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteFolder(folderId: number): Promise<ApiResponse> {
    return this.request(`/folders/${folderId}`, {
      method: 'DELETE',
    });
  }

  async markFolderAsRead(folderId: number): Promise<ApiResponse> {
    return this.request(`/folders/${folderId}/mark-read`, {
      method: 'PUT',
    });
  }

  async syncFolder(folderId: number): Promise<ApiResponse> {
    return this.request(`/folders/${folderId}/sync`, {
      method: 'PUT',
    });
  }

  async syncAccount(accountId: number): Promise<ApiResponse> {
    return this.request(`/accounts/${accountId}/sync`, {
      method: 'POST',
    });
  }

  async getEmailDetail(emailId: number): Promise<ApiResponse<Email>> {
    return this.request(`/emails/${emailId}`);
  }

  async markEmailAsRead(emailId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}/read`, {
      method: 'PUT',
    });
  }

  async markEmailAsUnread(emailId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}/unread`, {
      method: 'PUT',
    });
  }

  async toggleEmailStar(emailId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}/star`, {
      method: 'PUT',
    });
  }

  async deleteEmail(emailId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}`, {
      method: 'DELETE',
    });
  }

  async toggleEmailImportant(emailId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}`, {
      method: 'PATCH',
      body: JSON.stringify({ is_important: true }), // 后端会自动切换状态
    });
  }

  async downloadAttachment(attachmentId: number): Promise<Blob> {
    const response = await fetch(`${API_BASE_URL}/attachments/${attachmentId}/download`, {
      headers: {
        Authorization: `Bearer ${this.getAuthToken()}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to download attachment');
    }

    return response.blob();
  }

  async getEmail(id: number): Promise<ApiResponse<Email>> {
    return this.request(`/emails/${id}`);
  }

  async sendEmail(email: {
    account_id: number;
    to: EmailAddress[];
    cc?: EmailAddress[];
    bcc?: EmailAddress[];
    subject: string;
    text_body?: string;
    html_body?: string;
    attachment_ids?: number[];
    priority?: string;
    importance?: string;
    scheduled_time?: string;
    request_read_receipt?: boolean;
    request_delivery_receipt?: boolean;
  }): Promise<ApiResponse> {
    return this.request('/emails/send', {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async replyEmail(
    originalEmailId: number,
    email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
      attachment_ids?: number[];
      priority?: string;
      importance?: string;
      scheduled_time?: string;
      request_read_receipt?: boolean;
      request_delivery_receipt?: boolean;
    }
  ): Promise<ApiResponse> {
    return this.request(`/emails/${originalEmailId}/reply`, {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async replyAllEmail(
    originalEmailId: number,
    email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
      attachment_ids?: number[];
      priority?: string;
      importance?: string;
      scheduled_time?: string;
      request_read_receipt?: boolean;
      request_delivery_receipt?: boolean;
    }
  ): Promise<ApiResponse> {
    return this.request(`/emails/${originalEmailId}/reply-all`, {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async forwardEmail(
    originalEmailId: number,
    email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
      priority?: string;
      importance?: string;
      scheduled_time?: string;
      request_read_receipt?: boolean;
      request_delivery_receipt?: boolean;
    }
  ): Promise<ApiResponse> {
    return this.request(`/emails/${originalEmailId}/forward`, {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async archiveEmail(emailId: number): Promise<ApiResponse> {
    return this.request(`/emails/${emailId}/archive`, {
      method: 'PUT',
    });
  }

  async saveDraft(draft: {
    accountId?: number;
    to: string[];
    cc?: string[];
    bcc?: string[];
    subject: string;
    content?: string;
    htmlContent?: string;
    attachments?: unknown[];
  }): Promise<ApiResponse> {
    return this.request('/emails/draft', {
      method: 'POST',
      body: JSON.stringify({
        account_id: draft.accountId,
        to: draft.to.map((email) => ({ address: email })),
        cc: draft.cc?.map((email) => ({ address: email })) || [],
        bcc: draft.bcc?.map((email) => ({ address: email })) || [],
        subject: draft.subject,
        text_body: draft.content,
        html_body: draft.htmlContent,
      }),
    });
  }
}

export const apiClient = new ApiClient();
