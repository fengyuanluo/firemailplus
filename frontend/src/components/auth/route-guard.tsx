/**
 * 统一的路由守卫组件
 * 处理所有的路由保护和重定向逻辑
 */

'use client';

import { useEffect, ReactNode } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, useUIStore } from '@/lib/store';
import { useHydration } from '@/hooks/use-hydration';
import { HydrationLoader } from '@/components/ui/hydration-loader';

// 路由守卫配置
interface RouteGuardConfig {
  requireAuth?: boolean; // 是否需要认证
  mobileOnly?: boolean; // 是否仅限移动端
  desktopOnly?: boolean; // 是否仅限桌面端
  redirectTo?: string; // 自定义重定向路径
  allowUnauthenticated?: boolean; // 是否允许未认证用户访问
}

interface RouteGuardProps {
  children: ReactNode;
  config?: RouteGuardConfig;
}

// 默认配置
const DEFAULT_CONFIG: RouteGuardConfig = {
  requireAuth: true,
  mobileOnly: false,
  desktopOnly: false,
  allowUnauthenticated: false,
};

export function RouteGuard({ children, config = {} }: RouteGuardProps) {
  const finalConfig = { ...DEFAULT_CONFIG, ...config };
  const router = useRouter();
  const pathname = usePathname();

  const { isAuthenticated } = useAuthStore();
  const { isMobile, setIsMobile } = useUIStore();
  const { isHydrated } = useHydration();

  // 检测移动端状态
  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < 768;
      if (isMobile !== mobile) {
        setIsMobile(mobile);
      }
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, [isMobile, setIsMobile]);

  // 路由保护逻辑
  useEffect(() => {
    // 只在水合完成后执行重定向逻辑
    if (!isHydrated) return;

    // 1. 认证检查
    if (finalConfig.requireAuth && !isAuthenticated) {
      const redirectPath = finalConfig.redirectTo || '/login';
      if (pathname !== redirectPath) {
        router.replace(redirectPath);
        return;
      }
    }

    // 2. 已认证用户访问登录页面，重定向到邮箱
    if (isAuthenticated && pathname === '/login') {
      router.replace('/mailbox');
      return;
    }

    // 3. 设备类型检查
    if (finalConfig.mobileOnly && !isMobile) {
      // 仅限移动端的页面，桌面端用户重定向
      const redirectPath = finalConfig.redirectTo || '/mailbox';
      if (pathname !== redirectPath) {
        router.replace(redirectPath);
        return;
      }
    }

    if (finalConfig.desktopOnly && isMobile) {
      // 仅限桌面端的页面，移动端用户重定向
      const redirectPath = finalConfig.redirectTo || '/mailbox/mobile';
      if (pathname !== redirectPath) {
        router.replace(redirectPath);
        return;
      }
    }

    // 4. 移动端自动重定向逻辑
    if (
      isMobile &&
      !pathname.includes('/mobile') &&
      !pathname.includes('/search') &&
      !pathname.includes('/login')
    ) {
      // 移动端用户访问桌面端页面，重定向到移动端
      if (pathname === '/mailbox') {
        router.replace('/mailbox/mobile');
        return;
      }
    }

    // 5. 桌面端自动重定向逻辑
    if (!isMobile && pathname.includes('/mobile')) {
      // 桌面端用户访问移动端页面，重定向到桌面端
      const desktopPath = pathname.replace('/mobile', '');
      router.replace(desktopPath || '/mailbox');
      return;
    }
  }, [isHydrated, isAuthenticated, isMobile, pathname, router, finalConfig]);

  // 水合状态检查
  if (!isHydrated) {
    return <HydrationLoader message="正在初始化应用..." />;
  }

  // 认证状态检查
  if (finalConfig.requireAuth && !isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-900 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600">正在验证身份...</p>
        </div>
      </div>
    );
  }

  // 设备类型检查
  if (finalConfig.mobileOnly && !isMobile) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-900 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600">正在重定向...</p>
        </div>
      </div>
    );
  }

  if (finalConfig.desktopOnly && isMobile) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-900 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600">正在重定向...</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}

// 预定义的路由守卫配置
export const RouteConfigs = {
  // 公开页面（不需要认证）
  public: {
    requireAuth: false,
    allowUnauthenticated: true,
  },

  // 认证页面（已认证用户会被重定向）
  auth: {
    requireAuth: false,
    allowUnauthenticated: true,
  },

  // 受保护的页面（需要认证）
  protected: {
    requireAuth: true,
  },

  // 仅限移动端
  mobileOnly: {
    requireAuth: true,
    mobileOnly: true,
  },

  // 仅限桌面端
  desktopOnly: {
    requireAuth: true,
    desktopOnly: true,
  },
} as const;

// 便捷的路由守卫组件
export function PublicRoute({ children }: { children: ReactNode }) {
  return <RouteGuard config={RouteConfigs.public}>{children}</RouteGuard>;
}

export function AuthRoute({ children }: { children: ReactNode }) {
  return <RouteGuard config={RouteConfigs.auth}>{children}</RouteGuard>;
}

export function ProtectedRoute({ children }: { children: ReactNode }) {
  return <RouteGuard config={RouteConfigs.protected}>{children}</RouteGuard>;
}

export function MobileOnlyRoute({ children }: { children: ReactNode }) {
  return <RouteGuard config={RouteConfigs.mobileOnly}>{children}</RouteGuard>;
}

export function DesktopOnlyRoute({ children }: { children: ReactNode }) {
  return <RouteGuard config={RouteConfigs.desktopOnly}>{children}</RouteGuard>;
}
