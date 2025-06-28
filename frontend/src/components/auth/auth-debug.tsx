'use client';

import { useAuthStore } from '@/lib/store';

export function AuthDebug() {
  const { user, token, isAuthenticated, isHydrated } = useAuthStore();

  if (process.env.NODE_ENV !== 'development') {
    return null;
  }

  return (
    <div className="fixed bottom-4 right-4 bg-black text-white p-4 rounded-lg text-xs max-w-sm z-50">
      <h3 className="font-bold mb-2">Auth Debug</h3>
      <div className="space-y-1">
        <div>Hydrated: {isHydrated ? '✅' : '❌'}</div>
        <div>Authenticated: {isAuthenticated ? '✅' : '❌'}</div>
        <div>Token: {token ? `${token.substring(0, 10)}...` : 'None'}</div>
        <div>User: {user ? user.username : 'None'}</div>
        <div>
          LocalStorage Token:{' '}
          {typeof window !== 'undefined'
            ? localStorage.getItem('auth_token')
              ? 'Present'
              : 'None'
            : 'SSR'}
        </div>
      </div>
    </div>
  );
}
