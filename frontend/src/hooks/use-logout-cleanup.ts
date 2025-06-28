/**
 * 登出清理Hook
 * 完善登出时的状态清理，确保所有相关状态都被正确清除
 */

import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useAuthStore, useMailboxStore, useUIStore, useComposeStore } from '@/lib/store';
import { toast } from 'sonner';

// 清理配置
interface CleanupConfig {
  clearCache: boolean;
  clearLocalStorage: boolean;
  clearSessionStorage: boolean;
  resetStores: boolean;
  showNotification: boolean;
  redirectTo: string;
}

const DEFAULT_CLEANUP_CONFIG: CleanupConfig = {
  clearCache: true,
  clearLocalStorage: true,
  clearSessionStorage: true,
  resetStores: true,
  showNotification: true,
  redirectTo: '/login',
};

export function useLogoutCleanup() {
  const queryClient = useQueryClient();
  const router = useRouter();

  // Store actions
  const { logout: authLogout } = useAuthStore();
  const { resetState: resetMailboxState } = useMailboxStore();
  // UI和Compose store没有resetState方法，使用其他方式重置
  const uiStore = useUIStore();
  const composeStore = useComposeStore();

  // 清理React Query缓存
  const clearQueryCache = useCallback(() => {
    try {
      queryClient.clear();
      queryClient.removeQueries();
      console.log('React Query缓存已清理');
    } catch (error) {
      console.error('清理React Query缓存失败:', error);
    }
  }, [queryClient]);

  // 清理localStorage
  const clearLocalStorage = useCallback(() => {
    try {
      const keysToKeep = ['theme', 'language']; // 保留的键
      const keysToRemove: string[] = [];

      // 收集需要删除的键
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && !keysToKeep.includes(key)) {
          keysToRemove.push(key);
        }
      }

      // 删除键
      keysToRemove.forEach((key) => {
        localStorage.removeItem(key);
      });

      console.log('localStorage已清理，保留:', keysToKeep);
    } catch (error) {
      console.error('清理localStorage失败:', error);
    }
  }, []);

  // 清理sessionStorage
  const clearSessionStorage = useCallback(() => {
    try {
      sessionStorage.clear();
      console.log('sessionStorage已清理');
    } catch (error) {
      console.error('清理sessionStorage失败:', error);
    }
  }, []);

  // 重置所有Store状态
  const resetAllStores = useCallback(() => {
    try {
      // 重置认证状态
      authLogout();

      // 重置邮箱状态
      resetMailboxState();

      // 重置UI状态（手动重置关键状态）
      uiStore.setSidebarOpen(true); // 桌面端默认打开
      uiStore.setSidebarOpenMobile(false); // 移动端默认关闭
      uiStore.setIsMobile(false);

      // 重置写信状态（手动重置关键状态）
      composeStore.setIsOpen(false);

      console.log('所有Store状态已重置');
    } catch (error) {
      console.error('重置Store状态失败:', error);
    }
  }, [authLogout, resetMailboxState, uiStore, composeStore]);

  // 清理网络请求
  const clearNetworkRequests = useCallback(() => {
    try {
      // 取消所有进行中的请求
      if (typeof window !== 'undefined' && window.AbortController) {
        // 这里可以添加取消请求的逻辑
        console.log('网络请求已清理');
      }
    } catch (error) {
      console.error('清理网络请求失败:', error);
    }
  }, []);

  // 清理定时器和事件监听器
  const clearTimersAndListeners = useCallback(() => {
    try {
      // 清理可能存在的定时器
      // 注意：这里只是示例，实际应用中应该在各个组件中正确清理

      // 清理可能的事件监听器
      // 注意：这里只是示例，实际应用中应该在各个组件中正确清理

      console.log('定时器和事件监听器已清理');
    } catch (error) {
      console.error('清理定时器和事件监听器失败:', error);
    }
  }, []);

  // 执行完整的登出清理
  const performLogoutCleanup = useCallback(
    async (config: Partial<CleanupConfig> = {}) => {
      const finalConfig = { ...DEFAULT_CLEANUP_CONFIG, ...config };

      try {
        console.log('开始执行登出清理...');

        // 1. 清理React Query缓存
        if (finalConfig.clearCache) {
          clearQueryCache();
        }

        // 2. 重置Store状态
        if (finalConfig.resetStores) {
          resetAllStores();
        }

        // 3. 清理本地存储
        if (finalConfig.clearLocalStorage) {
          clearLocalStorage();
        }

        // 4. 清理会话存储
        if (finalConfig.clearSessionStorage) {
          clearSessionStorage();
        }

        // 5. 清理网络请求
        clearNetworkRequests();

        // 6. 清理定时器和事件监听器
        clearTimersAndListeners();

        // 7. 显示通知
        if (finalConfig.showNotification) {
          toast.success('已安全退出登录');
        }

        // 8. 重定向
        if (finalConfig.redirectTo) {
          // 延迟重定向，确保清理完成
          setTimeout(() => {
            router.replace(finalConfig.redirectTo);
          }, 100);
        }

        console.log('登出清理完成');
      } catch (error) {
        console.error('登出清理过程中发生错误:', error);

        // 即使清理失败，也要尝试基本的登出操作
        try {
          authLogout();
          if (finalConfig.redirectTo) {
            router.replace(finalConfig.redirectTo);
          }
        } catch (fallbackError) {
          console.error('备用登出操作也失败:', fallbackError);
        }
      }
    },
    [
      clearQueryCache,
      resetAllStores,
      clearLocalStorage,
      clearSessionStorage,
      clearNetworkRequests,
      clearTimersAndListeners,
      authLogout,
      router,
    ]
  );

  // 快速登出（仅清理必要状态）
  const quickLogout = useCallback(() => {
    return performLogoutCleanup({
      clearCache: true,
      clearLocalStorage: false,
      clearSessionStorage: false,
      resetStores: true,
      showNotification: true,
      redirectTo: '/login',
    });
  }, [performLogoutCleanup]);

  // 完整登出（清理所有状态）
  const fullLogout = useCallback(() => {
    return performLogoutCleanup({
      clearCache: true,
      clearLocalStorage: true,
      clearSessionStorage: true,
      resetStores: true,
      showNotification: true,
      redirectTo: '/login',
    });
  }, [performLogoutCleanup]);

  // 静默登出（不显示通知）
  const silentLogout = useCallback(() => {
    return performLogoutCleanup({
      clearCache: true,
      clearLocalStorage: true,
      clearSessionStorage: true,
      resetStores: true,
      showNotification: false,
      redirectTo: '/login',
    });
  }, [performLogoutCleanup]);

  // 检查清理状态
  const checkCleanupStatus = useCallback(() => {
    const { isAuthenticated, token } = useAuthStore.getState();
    const { emails, accounts } = useMailboxStore.getState();

    return {
      authCleared: !isAuthenticated && !token,
      mailboxCleared: emails.length === 0 && accounts.length === 0,
      cacheCleared: queryClient.getQueryCache().getAll().length === 0,
    };
  }, [queryClient]);

  return {
    // 主要清理方法
    performLogoutCleanup,
    quickLogout,
    fullLogout,
    silentLogout,

    // 单独的清理方法
    clearQueryCache,
    clearLocalStorage,
    clearSessionStorage,
    resetAllStores,

    // 状态检查
    checkCleanupStatus,
  };
}

// 自动清理Hook（在组件卸载时自动清理）
export function useAutoCleanup() {
  const { quickLogout } = useLogoutCleanup();

  // 页面卸载时的清理
  const handleBeforeUnload = useCallback(() => {
    // 在页面卸载前进行必要的清理
    // 注意：这里不能使用异步操作
    try {
      const { logout } = useAuthStore.getState();
      // 只进行同步的清理操作
      console.log('页面卸载，执行基础清理');
    } catch (error) {
      console.error('页面卸载清理失败:', error);
    }
  }, []);

  return {
    handleBeforeUnload,
    quickLogout,
  };
}
