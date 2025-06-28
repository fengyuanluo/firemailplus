'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/store';
import { useHydration } from '@/hooks/use-hydration';
import { PublicRoute } from '@/components/auth/route-guard';

export default function HomePage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const { isHydrated } = useHydration();

  useEffect(() => {
    // 只在水合完成后执行重定向逻辑
    if (!isHydrated) return;

    // 根据认证状态重定向
    if (isAuthenticated) {
      router.replace('/mailbox');
    } else {
      router.replace('/login');
    }
  }, [isAuthenticated, isHydrated, router]);

  return (
    <PublicRoute>
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-900 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600">{!isHydrated ? '正在初始化应用...' : '正在跳转...'}</p>
        </div>
      </div>
    </PublicRoute>
  );
}
