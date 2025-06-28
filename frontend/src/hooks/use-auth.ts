/**
 * 统一的认证管理Hook
 * 封装所有认证相关的逻辑，包括登录、登出、token验证等
 */

import { useEffect, useCallback } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { apiClient, type LoginRequest } from '@/lib/api';
import { useAuthStore } from '@/lib/store';
import { useHydration } from './use-hydration';
import { useTokenValidation } from './use-token-validation';
import { useLogoutCleanup } from './use-logout-cleanup';

// 认证错误类型
export enum AuthErrorType {
  INVALID_CREDENTIALS = 'INVALID_CREDENTIALS',
  TOKEN_EXPIRED = 'TOKEN_EXPIRED',
  NETWORK_ERROR = 'NETWORK_ERROR',
  UNKNOWN_ERROR = 'UNKNOWN_ERROR',
}

// 认证状态类型
export interface AuthStatus {
  isAuthenticated: boolean;
  isHydrated: boolean;
  isLoading: boolean;
  user: any | null;
  error: string | null;
}

export function useAuth() {
  const { user, isAuthenticated, isHydrated, login, logout } = useAuthStore();
  const { isHydrated: hydrationComplete } = useHydration();
  const { validateToken, isValidating } = useTokenValidation();
  const { fullLogout, quickLogout } = useLogoutCleanup();
  const router = useRouter();
  const queryClient = useQueryClient();

  // 安全的登出函数
  const performLogout = useCallback(
    async (showMessage = true) => {
      try {
        if (showMessage) {
          await fullLogout();
        } else {
          await quickLogout();
        }
      } catch (error) {
        console.error('Logout error:', error);
        // 备用登出逻辑
        logout();
        router.replace('/login');
      }
    },
    [fullLogout, quickLogout, logout, router]
  );

  // 登录 mutation
  const loginMutation = useMutation({
    mutationFn: (credentials: LoginRequest) => apiClient.login(credentials),
    onSuccess: (response) => {
      if (response.success && response.data) {
        login(response.data.user, response.data.token);
        toast.success('登录成功');
        router.push('/mailbox');
      } else {
        throw new Error(response.message || '登录失败');
      }
    },
    onError: (error: any) => {
      const errorMessage = error.message || '登录失败';
      toast.error(errorMessage);
      console.error('Login error:', error);
    },
  });

  // 登出 mutation
  const logoutMutation = useMutation({
    mutationFn: () => apiClient.logout(),
    onSuccess: () => {
      performLogout(true);
    },
    onError: (error: any) => {
      // 即使 API 调用失败，也要清除本地状态
      console.error('Logout API error:', error);
      performLogout(false);
      toast.error('退出登录失败，但已清除本地状态');
    },
  });

  // 获取当前用户信息
  const {
    data: currentUser,
    isLoading: isLoadingUser,
    error: userError,
  } = useQuery({
    queryKey: ['currentUser'],
    queryFn: async () => {
      try {
        const response = await apiClient.getCurrentUser();
        if (response.success && response.data) {
          return response.data;
        }
        throw new Error(response.message || '获取用户信息失败');
      } catch (error: any) {
        // 根据错误类型进行分类处理
        if (error.status === 401) {
          throw new Error('TOKEN_EXPIRED');
        }
        throw error;
      }
    },
    enabled: isAuthenticated && hydrationComplete,
    retry: (failureCount, error: any) => {
      // 如果是token过期，不重试
      if (error.message === 'TOKEN_EXPIRED') {
        return false;
      }
      // 其他错误最多重试1次
      return failureCount < 1;
    },
    staleTime: 5 * 60 * 1000, // 5分钟内不重新获取
    gcTime: 10 * 60 * 1000, // 10分钟后清除缓存
  });

  // 处理token验证失败
  useEffect(() => {
    if (userError && isAuthenticated && hydrationComplete) {
      const errorMessage = userError.message || '';

      if (errorMessage === 'TOKEN_EXPIRED' || errorMessage.includes('401')) {
        console.log('Token expired, performing automatic logout');
        performLogout(false);
        toast.error('登录已过期，请重新登录');
      } else {
        console.error('User validation error:', userError);
        // 对于其他错误，不自动登出，但显示错误信息
        toast.error('获取用户信息失败');
      }
    }
  }, [userError, isAuthenticated, hydrationComplete, performLogout]);

  // 检查认证状态
  const checkAuthStatus = useCallback((): AuthStatus => {
    return {
      isAuthenticated,
      isHydrated: hydrationComplete,
      isLoading: isLoadingUser || loginMutation.isPending || logoutMutation.isPending,
      user: currentUser || user,
      error: userError?.message || null,
    };
  }, [
    isAuthenticated,
    hydrationComplete,
    isLoadingUser,
    loginMutation.isPending,
    logoutMutation.isPending,
    currentUser,
    user,
    userError,
  ]);

  // 强制刷新用户信息
  const refreshUser = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['currentUser'] });
  }, [queryClient]);

  return {
    // 基础状态
    user: currentUser || user,
    isAuthenticated,
    isHydrated: hydrationComplete,
    isLoadingUser,

    // 操作方法
    login: loginMutation.mutate,
    logout: logoutMutation.mutate,
    performLogout,
    refreshUser,
    checkAuthStatus,
    validateToken,

    // 加载状态
    isLoggingIn: loginMutation.isPending,
    isLoggingOut: logoutMutation.isPending,
    isValidating,
    isLoading: isLoadingUser || loginMutation.isPending || logoutMutation.isPending || isValidating,

    // 错误状态
    error: userError?.message || null,
    loginError: loginMutation.error?.message || null,
    logoutError: logoutMutation.error?.message || null,
  };
}

// 简化的认证状态Hook（仅用于状态检查）
export function useAuthStatus() {
  const { isAuthenticated, isHydrated } = useAuthStore();
  const { isHydrated: hydrationComplete } = useHydration();

  return {
    isAuthenticated,
    isHydrated: hydrationComplete,
    isReady: hydrationComplete && isAuthenticated,
  };
}

// 认证守卫Hook（用于组件级别的认证检查）
export function useAuthGuard(redirectTo = '/login') {
  const { isAuthenticated, isHydrated } = useAuthStatus();
  const router = useRouter();

  useEffect(() => {
    if (isHydrated && !isAuthenticated) {
      router.replace(redirectTo);
    }
  }, [isHydrated, isAuthenticated, redirectTo, router]);

  return {
    isAuthenticated,
    isHydrated,
    shouldRender: isHydrated && isAuthenticated,
  };
}
