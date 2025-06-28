'use client';

/**
 * 全局 Providers 组件
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { Toaster } from '@/components/ui/sonner';
import { ThemeProvider } from 'next-themes';
import { AuthGuard } from '@/components/auth/auth-guard';
import { useState } from 'react';
import { handleError } from './error-handler';

interface ProvidersProps {
  children: React.ReactNode;
}

export function Providers({ children }: ProvidersProps) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // 缓存策略优化
            staleTime: 1000 * 60 * 5, // 5分钟内数据被认为是新鲜的
            gcTime: 1000 * 60 * 30, // 30分钟后清理未使用的缓存

            // 重试策略优化
            retry: (failureCount, error: any) => {
              // 认证错误不重试
              if (error?.status === 401 || error?.status === 403) {
                return false;
              }
              // 客户端错误（4xx）不重试
              if (error?.status >= 400 && error?.status < 500) {
                return false;
              }
              // 最多重试3次
              return failureCount < 3;
            },
            retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000), // 指数退避，最大30秒

            // 网络优化
            refetchOnWindowFocus: false, // 窗口聚焦时不自动重新获取
            refetchOnReconnect: true, // 网络重连时重新获取
            refetchOnMount: true, // 组件挂载时重新获取

            // 错误处理移到组件级别
          },
          mutations: {
            // 变更重试策略
            retry: (failureCount, error: any) => {
              // 认证错误不重试
              if (error?.status === 401 || error?.status === 403) {
                return false;
              }
              // 客户端错误不重试
              if (error?.status >= 400 && error?.status < 500) {
                return false;
              }
              // 网络错误重试1次
              return failureCount < 1;
            },
            retryDelay: 1000, // 1秒后重试

            // 错误处理
            onError: (error: any) => {
              handleError(error, 'react_query_mutation');
            },
          },
        },
      })
  );

  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={queryClient}>
        <AuthGuard>{children}</AuthGuard>
        <Toaster position="top-right" />
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
