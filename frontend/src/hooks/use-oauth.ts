/**
 * OAuth2 认证相关 Hooks
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';

interface OAuth2WindowOptions {
  width?: number;
  height?: number;
  left?: number;
  top?: number;
}

interface OAuth2CallbackData {
  code?: string;
  state?: string;
  error?: string;
  error_description?: string;
}

export function useOAuth2() {
  const { addAccount } = useMailboxStore();
  const queryClient = useQueryClient();

  // Gmail OAuth2 认证
  const gmailOAuthMutation = useMutation({
    mutationFn: async (callbackUrl?: string) => {
      const response = await apiClient.getGmailOAuthUrl(callbackUrl);
      if (response.success && response.data) {
        return response.data;
      }
      throw new Error(response.message || '获取Gmail授权URL失败');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Gmail OAuth2初始化失败');
    },
  });

  // Outlook OAuth2 认证
  const outlookOAuthMutation = useMutation({
    mutationFn: async (callbackUrl?: string) => {
      const response = await apiClient.getOutlookOAuthUrl(callbackUrl);
      if (response.success && response.data) {
        return response.data;
      }
      throw new Error(response.message || '获取Outlook授权URL失败');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Outlook OAuth2初始化失败');
    },
  });

  // 创建OAuth2账户
  const createOAuth2AccountMutation = useMutation({
    mutationFn: (account: {
      name: string;
      email: string;
      provider: string;
      access_token: string;
      refresh_token: string; // 必需，用于token验证和刷新
      expires_at: number;
      scope?: string;
      client_id: string; // 必需，用于token刷新
    }) => apiClient.createOAuth2Account(account),
    onSuccess: (response) => {
      if (response.success && response.data) {
        addAccount(response.data);
        queryClient.invalidateQueries({ queryKey: ['emailAccounts'] });
        toast.success('OAuth2邮箱账户创建成功');
      }
    },
    onError: (error: any) => {
      toast.error(error.message || '创建OAuth2账户失败');
    },
  });

  // 创建手动OAuth2账户
  const createManualOAuth2AccountMutation = useMutation({
    mutationFn: (account: {
      name: string;
      email: string;
      provider: string;
      client_id: string;
      client_secret?: string;
      refresh_token: string;
      scope?: string;
      auth_url?: string;
      token_url?: string;
    }) => apiClient.createManualOAuth2Account(account),
    onSuccess: (response) => {
      if (response.success && response.data) {
        addAccount(response.data);
        queryClient.invalidateQueries({ queryKey: ['emailAccounts'] });
        toast.success('手动OAuth2账户创建成功');
      }
    },
    onError: (error: any) => {
      toast.error(error.message || '创建手动OAuth2账户失败');
    },
  });

  // 打开OAuth2授权窗口
  const openOAuth2Window = (
    authUrl: string,
    options: OAuth2WindowOptions = {}
  ): Promise<OAuth2CallbackData> => {
    return new Promise((resolve, reject) => {
      const {
        width = 500,
        height = 600,
        left = window.screen.width / 2 - 250,
        top = window.screen.height / 2 - 300,
      } = options;

      const popup = window.open(
        authUrl,
        'oauth2_popup',
        `width=${width},height=${height},left=${left},top=${top},scrollbars=yes,resizable=yes`
      );

      if (!popup) {
        reject(new Error('无法打开授权窗口，请检查浏览器弹窗设置'));
        return;
      }

      // 监听窗口关闭
      const checkClosed = setInterval(() => {
        if (popup.closed) {
          clearInterval(checkClosed);
          reject(new Error('用户取消了授权'));
        }
      }, 1000);

      // 监听消息
      const messageHandler = (event: MessageEvent) => {
        // 验证来源
        if (event.origin !== window.location.origin) {
          return;
        }

        if (event.data.type === 'oauth2_callback') {
          clearInterval(checkClosed);
          popup.close();
          window.removeEventListener('message', messageHandler);

          if (event.data.error) {
            reject(new Error(event.data.error_description || event.data.error));
          } else {
            resolve(event.data);
          }
        }
      };

      window.addEventListener('message', messageHandler);
    });
  };

  // Gmail OAuth2 完整流程 - 直接跳转模式
  const authenticateGmail = async (accountName: string, email: string) => {
    try {
      // 1. 将账户信息编码到回调URL中
      const accountInfo = encodeURIComponent(
        JSON.stringify({
          name: accountName,
          email: email,
          provider: 'gmail',
        })
      );
      const callbackUrl = `${window.location.origin}/oauth/callback?account_info=${accountInfo}`;

      // 2. 获取授权URL
      const authData = await gmailOAuthMutation.mutateAsync(callbackUrl);

      // 3. 直接跳转到外部OAuth服务器
      window.location.href = authData.auth_url;
    } catch (error: any) {
      toast.error(error.message || 'Gmail认证失败');
      throw error;
    }
  };

  // Outlook OAuth2 完整流程 - 直接跳转模式
  const authenticateOutlook = async (accountName: string, email: string) => {
    try {
      // 1. 将账户信息编码到回调URL中
      const accountInfo = encodeURIComponent(
        JSON.stringify({
          name: accountName,
          email: email,
          provider: 'outlook',
        })
      );
      const callbackUrl = `${window.location.origin}/oauth/callback?account_info=${accountInfo}`;

      // 2. 获取授权URL
      const authData = await outlookOAuthMutation.mutateAsync(callbackUrl);

      // 3. 直接跳转到外部OAuth服务器
      window.location.href = authData.auth_url;
    } catch (error: any) {
      toast.error(error.message || 'Outlook认证失败');
      throw error;
    }
  };

  return {
    // Mutations
    gmailOAuth: gmailOAuthMutation,
    outlookOAuth: outlookOAuthMutation,
    createOAuth2Account: createOAuth2AccountMutation,
    createManualOAuth2Account: createManualOAuth2AccountMutation,

    // Helper functions
    authenticateGmail,
    authenticateOutlook,
    openOAuth2Window,

    // Loading states
    isAuthenticating: gmailOAuthMutation.isPending || outlookOAuthMutation.isPending,
    isCreating:
      createOAuth2AccountMutation.isPending || createManualOAuth2AccountMutation.isPending,
  };
}
