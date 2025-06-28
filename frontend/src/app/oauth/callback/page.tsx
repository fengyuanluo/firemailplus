'use client';

import { useEffect, useState, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';
import { ProtectedRoute } from '@/components/auth/route-guard';

function OAuthCallbackContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { addAccount } = useMailboxStore();
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing');
  const [message, setMessage] = useState('正在处理OAuth认证...');

  useEffect(() => {
    const handleOAuthCallback = async () => {
      try {
        // 获取URL参数
        const code = searchParams.get('code');
        const state = searchParams.get('state');
        const error = searchParams.get('error');
        const errorDescription = searchParams.get('error_description');

        // 检查是否直接从URL参数获取token信息（新流程）
        const accessToken = searchParams.get('access_token');
        const refreshToken = searchParams.get('refresh_token');
        const tokenType = searchParams.get('token_type');
        const expiresIn = searchParams.get('expires_in');
        const scope = searchParams.get('scope');
        const clientId = searchParams.get('client_id');

        if (error) {
          throw new Error(errorDescription || error);
        }

        // 如果没有token信息，则检查code和state（旧流程）
        if (!accessToken && (!code || !state)) {
          throw new Error('缺少授权码或状态参数');
        }

        // 从URL参数获取账户信息
        const accountInfoParam = searchParams.get('account_info');
        let accountData: any;
        if (!accountInfoParam) {
          // 如果URL参数中没有账户信息，尝试从sessionStorage获取（向后兼容）
          const accountDataStr = sessionStorage.getItem('oauth_account_data');
          if (!accountDataStr) {
            throw new Error('未找到账户信息，请重新开始授权流程');
          }
          accountData = JSON.parse(accountDataStr);
        } else {
          // 从URL参数解码账户信息
          accountData = JSON.parse(decodeURIComponent(accountInfoParam));
        }

        // 注意：移除state验证，因为我们现在使用account_info参数传递账户信息
        // 外部OAuth服务器已经验证了state的有效性

        setMessage('正在处理访问令牌...');

        let tokenData;
        if (accessToken) {
          // 新流程：直接从URL参数获取token
          tokenData = {
            access_token: accessToken,
            refresh_token: refreshToken || '',
            token_type: tokenType || 'Bearer',
            expires_in: parseInt(expiresIn || '3600'),
            scope: scope || '',
            client_id: clientId || '',
          };
        } else {
          // 旧流程：通过后端API处理OAuth回调
          const tokenResponse = await apiClient.handleOAuth2Callback(
            accountData.provider,
            code!,
            state!
          );

          if (!tokenResponse.success || !tokenResponse.data) {
            throw new Error(tokenResponse.message || 'Token交换失败');
          }
          tokenData = tokenResponse.data;
        }

        setMessage('正在创建邮箱账户...');

        // 验证必需的字段
        if (!tokenData.refresh_token) {
          throw new Error('缺少refresh_token，无法创建账户');
        }

        const tokenClientId = 'client_id' in tokenData ? tokenData.client_id : '';
        if (!tokenClientId) {
          throw new Error('缺少client_id，无法创建账户');
        }

        // 创建邮箱账户
        const createResponse = await apiClient.createOAuth2Account({
          name: accountData.name,
          email: accountData.email,
          provider: accountData.provider,
          access_token: tokenData.access_token,
          refresh_token: tokenData.refresh_token,
          expires_at: Date.now() + tokenData.expires_in * 1000,
          scope: tokenData.scope,
          client_id: tokenClientId,
        });

        if (createResponse.success && createResponse.data) {
          addAccount(createResponse.data);
          setStatus('success');
          setMessage('邮箱账户创建成功！');
          const providerName = accountData.provider === 'gmail' ? 'Gmail' : 'Outlook';
          toast.success(`${providerName}账户添加成功`);

          // 清除临时数据
          sessionStorage.removeItem('oauth_account_data');

          // 3秒后跳转到邮箱页面
          setTimeout(() => {
            router.push('/mailbox');
          }, 3000);
        } else {
          throw new Error(createResponse.message || '创建邮箱账户失败');
        }
      } catch (error: any) {
        console.error('OAuth callback error:', error);
        setStatus('error');
        setMessage(error.message || 'OAuth认证失败');
        toast.error(error.message || 'OAuth认证失败');

        // 清除临时数据
        sessionStorage.removeItem('oauth_account_data');

        // 5秒后跳转回添加邮箱页面
        setTimeout(() => {
          router.push('/add-email');
        }, 5000);
      }
    };

    handleOAuthCallback();
  }, [searchParams, router, addAccount]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-white dark:bg-gray-900">
      <div className="text-center max-w-md mx-auto px-4">
        <div className="mb-8">
          {status === 'processing' && (
            <div className="w-16 h-16 border-4 border-gray-300 border-t-gray-900 rounded-full animate-spin mx-auto mb-4"></div>
          )}
          {status === 'success' && (
            <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-8 h-8 text-green-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
          )}
          {status === 'error' && (
            <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-8 h-8 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </div>
          )}
        </div>

        <h1 className="text-2xl font-light text-gray-900 dark:text-gray-100 mb-4">
          {status === 'processing' && 'OAuth 认证处理中'}
          {status === 'success' && '认证成功'}
          {status === 'error' && '认证失败'}
        </h1>

        <p className="text-gray-600 dark:text-gray-400 mb-6">{message}</p>

        {status === 'success' && (
          <p className="text-sm text-gray-500 dark:text-gray-400">3秒后自动跳转到仪表板...</p>
        )}

        {status === 'error' && (
          <p className="text-sm text-gray-500 dark:text-gray-400">5秒后自动跳转到添加邮箱页面...</p>
        )}
      </div>
    </div>
  );
}

export default function OAuthCallbackPage() {
  return (
    <ProtectedRoute>
      <Suspense
        fallback={
          <div className="min-h-screen flex items-center justify-center bg-white dark:bg-gray-900">
            <div className="text-center max-w-md mx-auto px-4">
              <div className="w-16 h-16 border-4 border-gray-300 border-t-gray-900 rounded-full animate-spin mx-auto mb-4"></div>
              <h1 className="text-2xl font-light text-gray-900 dark:text-gray-100 mb-4">
                加载中...
              </h1>
              <p className="text-gray-600 dark:text-gray-400">正在初始化OAuth认证...</p>
            </div>
          </div>
        }
      >
        <OAuthCallbackContent />
      </Suspense>
    </ProtectedRoute>
  );
}
