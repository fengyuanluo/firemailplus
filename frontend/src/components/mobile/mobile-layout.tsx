'use client';

import { NotificationCenter } from '@/components/mailbox/notification-center';
import { useNotificationStore } from '@/lib/store';

interface MobileLayoutProps {
  children: React.ReactNode;
  className?: string;
}

export function MobileLayout({ children, className = '' }: MobileLayoutProps) {
  const { notifications, removeNotification, markAsRead } = useNotificationStore();

  // 移动端重定向逻辑已移至 RouteGuard 组件

  return (
    <div className={`min-h-screen bg-gray-50 dark:bg-gray-900 ${className}`}>
      {children}

      {/* 通知中心 */}
      <NotificationCenter
        notifications={notifications}
        onRemove={removeNotification}
        onMarkAsRead={markAsRead}
      />
    </div>
  );
}

// 页面容器组件
interface MobilePageProps {
  children: React.ReactNode;
  className?: string;
}

export function MobilePage({ children, className = '' }: MobilePageProps) {
  return <div className={`flex flex-col h-screen ${className}`}>{children}</div>;
}

// 内容区域组件
interface MobileContentProps {
  children: React.ReactNode;
  className?: string;
  padding?: boolean;
}

export function MobileContent({ children, className = '', padding = true }: MobileContentProps) {
  return (
    <main
      className={`flex-1 overflow-y-auto bg-white dark:bg-gray-800 ${padding ? 'p-4' : ''} ${className}`}
    >
      {children}
    </main>
  );
}

// 列表容器组件
interface MobileListProps {
  children: React.ReactNode;
  className?: string;
}

export function MobileList({ children, className = '' }: MobileListProps) {
  return (
    <div className={`divide-y divide-gray-200 dark:divide-gray-700 ${className}`}>{children}</div>
  );
}

// 列表项组件
interface MobileListItemProps {
  children: React.ReactNode;
  onClick?: () => void;
  className?: string;
  active?: boolean;
}

export function MobileListItem({
  children,
  onClick,
  className = '',
  active = false,
}: MobileListItemProps) {
  return (
    <div
      onClick={onClick}
      className={`
        px-4 py-4 cursor-pointer transition-colors duration-150
        hover:bg-gray-50 dark:hover:bg-gray-750
        ${active ? 'bg-blue-50 dark:bg-blue-900/20 border-r-2 border-blue-500' : ''}
        ${className}
      `}
    >
      {children}
    </div>
  );
}

// 空状态组件
interface MobileEmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}

export function MobileEmptyState({ icon, title, description, action }: MobileEmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center h-full p-8 text-center">
      {icon && (
        <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mb-4">
          {icon}
        </div>
      )}

      <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">{title}</h3>

      {description && <p className="text-gray-500 dark:text-gray-400 mb-4">{description}</p>}

      {action}
    </div>
  );
}

// 加载状态组件
interface MobileLoadingProps {
  message?: string;
}

export function MobileLoading({ message = '加载中...' }: MobileLoadingProps) {
  return (
    <div className="flex flex-col items-center justify-center h-full p-8">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-4"></div>
      <p className="text-gray-500 dark:text-gray-400">{message}</p>
    </div>
  );
}
