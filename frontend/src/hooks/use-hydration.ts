/**
 * 统一的水合状态管理Hook
 * 提供一致的水合状态检查和加载UI
 */

import { useAuthStore } from '@/lib/store';

interface HydrationState {
  isHydrated: boolean;
  isLoading: boolean;
}

/**
 * 获取水合状态
 */
export function useHydration(): HydrationState {
  const { isHydrated } = useAuthStore();

  return {
    isHydrated,
    isLoading: !isHydrated,
  };
}

/**
 * 水合状态检查工具
 */
export function getHydrationStatus() {
  if (typeof window === 'undefined') {
    return { isHydrated: false, isLoading: true };
  }

  // 简单的水合状态检查
  const isHydrated = document.readyState === 'complete';
  return { isHydrated, isLoading: !isHydrated };
}

/**
 * 水合守卫Hook
 * 提供水合状态检查
 */
export function useHydrationGuard() {
  const { isHydrated } = useHydration();

  return {
    isHydrated,
    shouldRender: isHydrated,
  };
}
