/**
 * Token验证机制Hook
 * 优化token验证失败的处理逻辑，防止无限重定向
 */

import { useEffect, useRef, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

// Token验证状态
interface TokenValidationState {
  isValidating: boolean;
  lastValidation: number | null;
  validationCount: number;
  isValid: boolean | null;
}

// Token验证配置
interface TokenValidationConfig {
  validateInterval: number; // 验证间隔（毫秒）
  maxRetries: number; // 最大重试次数
  retryDelay: number; // 重试延迟（毫秒）
  silentValidation: boolean; // 是否静默验证
}

const DEFAULT_CONFIG: TokenValidationConfig = {
  validateInterval: 5 * 60 * 1000, // 5分钟
  maxRetries: 3,
  retryDelay: 1000,
  silentValidation: true,
};

export function useTokenValidation(config: Partial<TokenValidationConfig> = {}) {
  const finalConfig = { ...DEFAULT_CONFIG, ...config };
  const { token, isAuthenticated, logout } = useAuthStore();
  const router = useRouter();

  const validationState = useRef<TokenValidationState>({
    isValidating: false,
    lastValidation: null,
    validationCount: 0,
    isValid: null,
  });

  const redirectAttempts = useRef(0);
  const maxRedirectAttempts = 3;

  // 安全的重定向函数，防止无限重定向
  const safeRedirect = useCallback(
    (path: string) => {
      if (redirectAttempts.current >= maxRedirectAttempts) {
        console.warn('达到最大重定向次数，停止重定向');
        return;
      }

      redirectAttempts.current++;

      // 延迟重定向，避免在渲染期间调用
      setTimeout(() => {
        router.replace(path);
      }, 100);

      // 重置重定向计数器
      setTimeout(() => {
        redirectAttempts.current = 0;
      }, 5000);
    },
    [router]
  );

  // 验证token的核心函数
  const validateToken = useCallback(
    async (silent = true): Promise<boolean> => {
      if (!token || !isAuthenticated) {
        return false;
      }

      if (validationState.current.isValidating) {
        return validationState.current.isValid ?? false;
      }

      validationState.current.isValidating = true;
      validationState.current.validationCount++;

      try {
        const response = await apiClient.getCurrentUser();

        if (response.success && response.data) {
          validationState.current.isValid = true;
          validationState.current.lastValidation = Date.now();

          if (!silent) {
            console.log('Token验证成功');
          }

          return true;
        } else {
          throw new Error(response.message || 'Token验证失败');
        }
      } catch (error: any) {
        validationState.current.isValid = false;

        if (!silent) {
          console.error('Token验证失败:', error);
        }

        // 根据错误类型决定处理方式
        if (error.status === 401 || error.message?.includes('401')) {
          // Token过期或无效
          if (!silent) {
            toast.error('登录已过期，请重新登录');
          }

          // 清除认证状态
          logout();

          // 安全重定向到登录页
          safeRedirect('/login');

          return false;
        } else {
          // 网络错误或其他错误，不清除认证状态
          if (!silent) {
            toast.error('网络连接异常，请检查网络');
          }

          return false;
        }
      } finally {
        validationState.current.isValidating = false;
      }
    },
    [token, isAuthenticated, logout, safeRedirect]
  );

  // 带重试的token验证
  const validateTokenWithRetry = useCallback(
    async (silent = true): Promise<boolean> => {
      let attempts = 0;

      while (attempts < finalConfig.maxRetries) {
        const isValid = await validateToken(silent);

        if (isValid || !isAuthenticated) {
          return isValid;
        }

        attempts++;

        if (attempts < finalConfig.maxRetries) {
          // 等待后重试
          await new Promise((resolve) => setTimeout(resolve, finalConfig.retryDelay * attempts));
        }
      }

      return false;
    },
    [validateToken, finalConfig.maxRetries, finalConfig.retryDelay, isAuthenticated]
  );

  // 检查是否需要验证
  const shouldValidate = useCallback((): boolean => {
    if (!token || !isAuthenticated) {
      return false;
    }

    const now = Date.now();
    const lastValidation = validationState.current.lastValidation;

    // 如果从未验证过，需要验证
    if (!lastValidation) {
      return true;
    }

    // 如果距离上次验证超过间隔时间，需要验证
    return now - lastValidation > finalConfig.validateInterval;
  }, [token, isAuthenticated, finalConfig.validateInterval]);

  // 定期验证token
  useEffect(() => {
    if (!isAuthenticated || !token) {
      return;
    }

    const validatePeriodically = async () => {
      if (shouldValidate()) {
        await validateTokenWithRetry(finalConfig.silentValidation);
      }
    };

    // 立即验证一次
    validatePeriodically();

    // 设置定期验证
    const interval = setInterval(validatePeriodically, finalConfig.validateInterval);

    return () => clearInterval(interval);
  }, [
    isAuthenticated,
    token,
    shouldValidate,
    validateTokenWithRetry,
    finalConfig.validateInterval,
    finalConfig.silentValidation,
  ]);

  // 页面可见性变化时验证token
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && isAuthenticated && shouldValidate()) {
        validateTokenWithRetry(true);
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [isAuthenticated, shouldValidate, validateTokenWithRetry]);

  // 手动验证token
  const manualValidate = useCallback(() => {
    return validateTokenWithRetry(false);
  }, [validateTokenWithRetry]);

  // 获取验证状态
  const getValidationStatus = useCallback(() => {
    return {
      isValidating: validationState.current.isValidating,
      lastValidation: validationState.current.lastValidation,
      validationCount: validationState.current.validationCount,
      isValid: validationState.current.isValid,
      shouldValidate: shouldValidate(),
    };
  }, [shouldValidate]);

  return {
    validateToken: manualValidate,
    getValidationStatus,
    isValidating: validationState.current.isValidating,
    isValid: validationState.current.isValid,
  };
}

// 简化的token验证Hook（仅用于状态检查）
export function useTokenStatus() {
  const { token, isAuthenticated } = useAuthStore();

  return {
    hasToken: !!token,
    isAuthenticated,
    tokenExists: !!token && token.length > 0,
  };
}

// Token自动刷新Hook
export function useTokenRefresh() {
  const { token, login } = useAuthStore();
  const refreshAttempts = useRef(0);
  const maxRefreshAttempts = 3;

  const refreshToken = useCallback(async (): Promise<boolean> => {
    if (!token || refreshAttempts.current >= maxRefreshAttempts) {
      return false;
    }

    refreshAttempts.current++;

    try {
      // 这里应该调用refresh token API
      // const response = await apiClient.refreshToken();
      // if (response.success && response.data) {
      //   login(response.data.user, response.data.token);
      //   return true;
      // }

      console.log('Token refresh not implemented yet');
      return false;
    } catch (error) {
      console.error('Token refresh failed:', error);
      return false;
    } finally {
      // 重置刷新计数器
      setTimeout(() => {
        refreshAttempts.current = 0;
      }, 60000); // 1分钟后重置
    }
  }, [token, login]);

  return {
    refreshToken,
    canRefresh: refreshAttempts.current < maxRefreshAttempts,
  };
}
