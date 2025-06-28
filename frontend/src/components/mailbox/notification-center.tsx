'use client';

import { useEffect } from 'react';
import { X, Info, CheckCircle, AlertTriangle, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface Notification {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  autoClose?: boolean;
  duration?: number;
}

interface NotificationCenterProps {
  notifications: Notification[];
  onRemove: (id: string) => void;
  onMarkAsRead: (id: string) => void;
}

export function NotificationCenter({
  notifications,
  onRemove,
  onMarkAsRead,
}: NotificationCenterProps) {
  // 自动关闭通知
  useEffect(() => {
    notifications.forEach((notification) => {
      if (notification.autoClose && notification.duration) {
        const timer = setTimeout(() => {
          onRemove(notification.id);
        }, notification.duration);

        return () => clearTimeout(timer);
      }
    });
  }, [notifications, onRemove]);

  // 获取图标
  const getIcon = (type: Notification['type']) => {
    const iconClass = 'w-5 h-5 flex-shrink-0';

    switch (type) {
      case 'info':
        return <Info className={`${iconClass} text-blue-500`} />;
      case 'success':
        return <CheckCircle className={`${iconClass} text-green-500`} />;
      case 'warning':
        return <AlertTriangle className={`${iconClass} text-yellow-500`} />;
      case 'error':
        return <AlertCircle className={`${iconClass} text-red-500`} />;
      default:
        return <Info className={`${iconClass} text-gray-500`} />;
    }
  };

  // 获取样式类
  const getNotificationClasses = (type: Notification['type'], read: boolean) => {
    const baseClasses =
      'relative p-4 rounded-lg border shadow-sm transition-all duration-300 ease-in-out';
    const readClasses = read ? 'opacity-75' : '';

    const typeClasses = {
      info: 'bg-blue-50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-800',
      success: 'bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800',
      warning: 'bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800',
      error: 'bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800',
    };

    return `${baseClasses} ${typeClasses[type]} ${readClasses}`;
  };

  // 格式化时间
  const formatTime = (timestamp: number) => {
    const now = Date.now();
    const diff = now - timestamp;

    if (diff < 60000) {
      // 小于1分钟
      return '刚刚';
    } else if (diff < 3600000) {
      // 小于1小时
      return `${Math.floor(diff / 60000)}分钟前`;
    } else if (diff < 86400000) {
      // 小于1天
      return `${Math.floor(diff / 3600000)}小时前`;
    } else {
      return new Date(timestamp).toLocaleDateString('zh-CN');
    }
  };

  if (notifications.length === 0) {
    return null;
  }

  return (
    <div className="fixed top-4 right-4 z-50 w-96 max-w-[calc(100vw-2rem)] space-y-2">
      {notifications.slice(0, 5).map((notification) => (
        <div
          key={notification.id}
          className={getNotificationClasses(notification.type, notification.read)}
          onClick={() => !notification.read && onMarkAsRead(notification.id)}
        >
          {/* 关闭按钮 */}
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              onRemove(notification.id);
            }}
            className="absolute top-2 right-2 p-1 h-auto w-auto hover:bg-black/10 dark:hover:bg-white/10"
          >
            <X className="w-4 h-4" />
          </Button>

          {/* 通知内容 */}
          <div className="flex items-start gap-3 pr-8">
            {/* 图标 */}
            {getIcon(notification.type)}

            {/* 文本内容 */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between mb-1">
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                  {notification.title}
                </h4>
                {!notification.read && (
                  <div className="w-2 h-2 bg-blue-500 rounded-full flex-shrink-0 ml-2" />
                )}
              </div>

              <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
                {notification.message}
              </p>

              <div className="mt-2 text-xs text-gray-500 dark:text-gray-500">
                {formatTime(notification.timestamp)}
              </div>
            </div>
          </div>

          {/* 进度条（用于自动关闭的通知） */}
          {notification.autoClose && notification.duration && (
            <div className="absolute bottom-0 left-0 right-0 h-1 bg-gray-200 dark:bg-gray-700 rounded-b-lg overflow-hidden">
              <div
                className="h-full bg-current opacity-30 animate-pulse"
                style={{
                  animation: `shrink ${notification.duration}ms linear forwards`,
                }}
              />
            </div>
          )}
        </div>
      ))}

      {/* 显示更多通知的提示 */}
      {notifications.length > 5 && (
        <div className="text-center">
          <div className="inline-flex items-center px-3 py-2 text-xs text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm">
            还有 {notifications.length - 5} 条通知
          </div>
        </div>
      )}

      <style jsx>{`
        @keyframes shrink {
          from {
            width: 100%;
          }
          to {
            width: 0%;
          }
        }
      `}</style>
    </div>
  );
}

// 单个通知组件
interface NotificationItemProps {
  notification: Notification;
  onRemove: (id: string) => void;
  onMarkAsRead: (id: string) => void;
}

export function NotificationItem({ notification, onRemove, onMarkAsRead }: NotificationItemProps) {
  return (
    <div className="p-4 border-b border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-750 transition-colors">
      <div className="flex items-start gap-3">
        {/* 状态指示器 */}
        {!notification.read && (
          <div className="w-2 h-2 bg-blue-500 rounded-full flex-shrink-0 mt-2" />
        )}

        {/* 内容 */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-1">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
              {notification.title}
            </h4>
            <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0 ml-2">
              {new Date(notification.timestamp).toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
              })}
            </span>
          </div>

          <p className="text-sm text-gray-600 dark:text-gray-400">{notification.message}</p>
        </div>

        {/* 操作按钮 */}
        <div className="flex items-center gap-1 flex-shrink-0">
          {!notification.read && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onMarkAsRead(notification.id)}
              className="p-1 h-auto text-xs"
            >
              标记已读
            </Button>
          )}

          <Button
            variant="ghost"
            size="sm"
            onClick={() => onRemove(notification.id)}
            className="p-1 h-auto"
          >
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}

// 通知列表组件
interface NotificationListProps {
  notifications: Notification[];
  onRemove: (id: string) => void;
  onMarkAsRead: (id: string) => void;
  onMarkAllAsRead: () => void;
  onClearAll: () => void;
}

export function NotificationList({
  notifications,
  onRemove,
  onMarkAsRead,
  onMarkAllAsRead,
  onClearAll,
}: NotificationListProps) {
  const unreadCount = notifications.filter((n) => !n.read).length;

  return (
    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg max-h-96 overflow-hidden">
      {/* 头部 */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            通知 {unreadCount > 0 && `(${unreadCount})`}
          </h3>

          <div className="flex items-center gap-2">
            {unreadCount > 0 && (
              <Button variant="ghost" size="sm" onClick={onMarkAllAsRead} className="text-xs">
                全部已读
              </Button>
            )}

            <Button variant="ghost" size="sm" onClick={onClearAll} className="text-xs">
              清空
            </Button>
          </div>
        </div>
      </div>

      {/* 通知列表 */}
      <div className="max-h-80 overflow-y-auto">
        {notifications.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">暂无通知</div>
        ) : (
          notifications.map((notification) => (
            <NotificationItem
              key={notification.id}
              notification={notification}
              onRemove={onRemove}
              onMarkAsRead={onMarkAsRead}
            />
          ))
        )}
      </div>
    </div>
  );
}
