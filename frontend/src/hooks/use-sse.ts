/**
 * SSE Hook - 集成 SSE 客户端到 React 应用
 */

import { useEffect, useRef, useCallback, useState, useMemo } from 'react';
import { useAuthStore, useMailboxStore } from '@/lib/store';
import { FireMailSSEClient } from '@/lib/sse-client';
import { refreshEmailAccountsAndGroupsIntoStore } from '@/lib/mailbox-group-data';
import type {
  SSEClientState,
  SSEConnectionStats,
  SSEEventType,
  SSEEventHandler,
  NewEmailEventData,
  EmailStatusEventData,
  SyncEventData,
  NotificationEventData,
  GroupEventData,
  AccountGroupEventData,
} from '@/types/sse';
import type { Email } from '@/types/email';

interface UseSSEOptions {
  autoConnect?: boolean;
  onNewEmail?: (data: NewEmailEventData) => void;
  onEmailStatusChange?: (data: EmailStatusEventData) => void;
  onSyncEvent?: (data: SyncEventData) => void;
  onNotification?: (data: NotificationEventData) => void;
  onGroupEvent?: (eventType: SSEEventType, data: GroupEventData) => void;
  onAccountGroupEvent?: (data: AccountGroupEventData) => void;
}

interface UseSSEReturn {
  // 连接状态
  state: SSEClientState;
  isConnected: boolean;
  stats: SSEConnectionStats;

  // 连接控制
  connect: () => void;
  disconnect: () => void;
  reconnect: () => void;

  // 事件监听
  on: <T = unknown>(eventType: SSEEventType, handler: SSEEventHandler<T>) => void;
  off: <T = unknown>(eventType: SSEEventType, handler: SSEEventHandler<T>) => void;
}

export function useSSE(options: UseSSEOptions = {}): UseSSEReturn {
  const {
    autoConnect = true,
    onNewEmail,
    onEmailStatusChange,
    onSyncEvent,
    onNotification,
    onGroupEvent,
    onAccountGroupEvent,
  } = options;
  const { token, isAuthenticated } = useAuthStore();

  // 稳定化 options 对象，避免每次渲染都创建新的引用
  const stableOptions = useMemo(
    () => ({
      onNewEmail,
      onEmailStatusChange,
      onSyncEvent,
      onNotification,
      onGroupEvent,
      onAccountGroupEvent,
    }),
    [onNewEmail, onEmailStatusChange, onSyncEvent, onNotification, onGroupEvent, onAccountGroupEvent]
  );

  const clientRef = useRef<FireMailSSEClient | null>(null);
  const [state, setState] = useState<SSEClientState>('disconnected');
  const [stats, setStats] = useState<SSEConnectionStats>({
    state: 'disconnected',
    reconnectAttempts: 0,
    totalEvents: 0,
    eventsByType: {} as Record<SSEEventType, number>,
  });

  // 创建 SSE 客户端
  const createClient = useCallback(() => {
    if (!token) {
      console.warn('🔐 [useSSE] 无法创建 SSE 客户端：缺少认证令牌');
      return null;
    }

    const client = new FireMailSSEClient({
      baseUrl: process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080',
      token,
      autoReconnect: true,
      reconnectInterval: 3000,
      maxReconnectAttempts: 10,
      heartbeatTimeout: 60000,
    });

    // 监听状态变更
    client.onStateChange((newState) => {
      setState(newState);
      // 状态变更时也需要谨慎更新统计信息
      const newStats = client.getStats();
      setStats((prevStats) => {
        // 简化比较逻辑，避免频繁的JSON.stringify
        if (
          prevStats.state !== newStats.state ||
          prevStats.reconnectAttempts !== newStats.reconnectAttempts ||
          prevStats.totalEvents !== newStats.totalEvents
        ) {
          return newStats;
        }
        return prevStats;
      });
    });

    // 监听错误
    client.onError((error) => {
      console.error('❌ [useSSE] SSE 客户端错误:', error);
    });

    // 注册业务事件处理器
    if (stableOptions.onNewEmail) {
      client.on('new_email', (event) => {
        stableOptions.onNewEmail!(event.data as NewEmailEventData);
      });
    }

    if (stableOptions.onEmailStatusChange) {
      client.on('email_read', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
      client.on('email_unread', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
      client.on('email_starred', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
      client.on('email_unstarred', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
      client.on('email_important', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
      client.on('email_unimportant', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
      client.on('email_deleted', (event) => {
        stableOptions.onEmailStatusChange!(event.data as EmailStatusEventData);
      });
    }

    if (stableOptions.onSyncEvent) {
      client.on('sync_started', (event) => {
        stableOptions.onSyncEvent!(event.data as SyncEventData);
      });
      client.on('sync_progress', (event) => {
        stableOptions.onSyncEvent!(event.data as SyncEventData);
      });
      client.on('sync_completed', (event) => {
        stableOptions.onSyncEvent!(event.data as SyncEventData);
      });
      client.on('sync_error', (event) => {
        stableOptions.onSyncEvent!(event.data as SyncEventData);
      });
    }

    if (stableOptions.onNotification) {
      client.on('notification', (event) => {
        stableOptions.onNotification!(event.data as NotificationEventData);
      });
    }

    if (stableOptions.onGroupEvent) {
      const groupEventTypes: SSEEventType[] = [
        'group_created',
        'group_updated',
        'group_deleted',
        'group_reordered',
        'group_default_changed',
      ];
      groupEventTypes.forEach((eventType) => {
        client.on(eventType, (event) => {
          stableOptions.onGroupEvent!(eventType, event.data as GroupEventData);
        });
      });
    }

    if (stableOptions.onAccountGroupEvent) {
      client.on('account_group_changed', (event) => {
        stableOptions.onAccountGroupEvent!(event.data as AccountGroupEventData);
      });
    }

    return client;
  }, [token, stableOptions]);

  // 连接函数
  const connect = useCallback(() => {
    if (!isAuthenticated || !token) {
      console.warn('🔐 [useSSE] 无法连接：用户未认证');
      return;
    }

    if (clientRef.current) {
      clientRef.current.disconnect();
    }

    const client = createClient();
    if (client) {
      clientRef.current = client;
      client.connect();
      console.log('🔗 [useSSE] 开始连接 SSE 服务器');
    }
  }, [isAuthenticated, token, createClient]);

  // 断开连接函数
  const disconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
      clientRef.current = null;
      setState('disconnected');
      console.log('🔌 [useSSE] 已断开 SSE 连接');
    }
  }, []);

  // 重连函数
  const reconnect = useCallback(() => {
    disconnect();
    setTimeout(connect, 1000); // 延迟 1 秒重连
  }, [disconnect, connect]);

  // 事件监听函数
  const on = useCallback(<T = unknown>(eventType: SSEEventType, handler: SSEEventHandler<T>) => {
    if (clientRef.current) {
      clientRef.current.on(eventType, handler);
    }
  }, []);

  // 移除事件监听函数
  const off = useCallback(<T = unknown>(eventType: SSEEventType, handler: SSEEventHandler<T>) => {
    if (clientRef.current) {
      clientRef.current.off(eventType, handler);
    }
  }, []);

  // 监听认证状态变化
  useEffect(() => {
    if (isAuthenticated && token && autoConnect) {
      connect();
    } else {
      disconnect();
    }
  }, [isAuthenticated, token, autoConnect, connect, disconnect]);

  // 监听 token 变化
  useEffect(() => {
    if (clientRef.current && token) {
      clientRef.current.updateToken(token);
    }
  }, [token]);

  // 组件卸载时清理
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  // 定期更新统计信息 - 使用深度比较避免不必要的更新
  useEffect(() => {
    const interval = setInterval(() => {
      if (clientRef.current) {
        const newStats = clientRef.current.getStats();
        // 只有当统计信息真正发生变化时才更新状态
        setStats((prevStats) => {
          // 简化比较逻辑，避免频繁的JSON.stringify
          if (
            prevStats.state !== newStats.state ||
            prevStats.reconnectAttempts !== newStats.reconnectAttempts ||
            prevStats.totalEvents !== newStats.totalEvents
          ) {
            return newStats;
          }
          return prevStats; // 返回相同的引用，避免不必要的重渲染
        });
      }
    }, 10000); // 增加到每 10 秒更新一次，减少频率

    return () => clearInterval(interval);
  }, []);

  return {
    state,
    isConnected: state === 'connected',
    stats,
    connect,
    disconnect,
    reconnect,
    on,
    off,
  };
}

// 专用的邮箱 SSE Hook
export function useMailboxSSE() {
  const [newEmailCount, setNewEmailCount] = useState(0);
  const [syncStatus, setSyncStatus] = useState<Record<number, SyncEventData>>({});
  const updateEmail = useMailboxStore((state) => state.updateEmail);
  const removeEmail = useMailboxStore((state) => state.removeEmail);
  const groupRefreshTimerRef = useRef<NodeJS.Timeout | null>(null);
  const groupRefreshInFlightRef = useRef(false);

  // 稳定化事件处理器，避免每次渲染都创建新的函数
  const handleNewEmail = useCallback((data: NewEmailEventData) => {
    if (process.env.NODE_ENV === 'development') {
      console.log('📧 [useMailboxSSE] 收到新邮件:', data);
    }
    setNewEmailCount((prev) => prev + 1);

    // 显示桌面通知
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification(`新邮件: ${data.subject}`, {
        body: `来自: ${data.from}`,
        icon: '/favicon.ico',
        tag: `email-${data.email_id}`,
      });
    }
  }, []);

  const handleEmailStatusChange = useCallback((data: EmailStatusEventData) => {
    if (process.env.NODE_ENV === 'development') {
      console.log('📝 [useMailboxSSE] 邮件状态变更:', data);
    }
    if (data.is_deleted) {
      removeEmail(data.email_id);
      return;
    }

    const updates: Partial<Email> = {};
    if (typeof data.is_read === 'boolean') {
      updates.is_read = data.is_read;
    }
    if (typeof data.is_starred === 'boolean') {
      updates.is_starred = data.is_starred;
    }
    if (typeof data.is_important === 'boolean') {
      updates.is_important = data.is_important;
    }

    if (Object.keys(updates).length > 0) {
      updateEmail(data.email_id, updates);
    }
  }, [removeEmail, updateEmail]);

  const handleSyncEvent = useCallback((data: SyncEventData) => {
    if (process.env.NODE_ENV === 'development') {
      console.log('🔄 [useMailboxSSE] 同步事件:', data);
    }
    setSyncStatus((prev) => ({
      ...prev,
      [data.account_id]: data,
    }));
  }, []);

  const handleNotification = useCallback((data: NotificationEventData) => {
    if (process.env.NODE_ENV === 'development') {
      console.log('🔔 [useMailboxSSE] 系统通知:', data);
    }
    // 这里可以显示应用内通知
  }, []);

  const scheduleMailboxStructureRefresh = useCallback(() => {
    if (groupRefreshTimerRef.current) {
      clearTimeout(groupRefreshTimerRef.current);
    }

    groupRefreshTimerRef.current = setTimeout(async () => {
      if (groupRefreshInFlightRef.current) {
        return;
      }

      groupRefreshInFlightRef.current = true;
      try {
        await refreshEmailAccountsAndGroupsIntoStore();
      } catch (error) {
        console.error('❌ [useMailboxSSE] 刷新邮箱分组结构失败:', error);
      } finally {
        groupRefreshInFlightRef.current = false;
      }
    }, 250);
  }, []);

  const handleGroupEvent = useCallback(
    (eventType: SSEEventType, data: GroupEventData) => {
      if (process.env.NODE_ENV === 'development') {
        console.log('🗂️ [useMailboxSSE] 分组事件:', eventType, data);
      }
      scheduleMailboxStructureRefresh();
    },
    [scheduleMailboxStructureRefresh]
  );

  const handleAccountGroupEvent = useCallback(
    (data: AccountGroupEventData) => {
      if (process.env.NODE_ENV === 'development') {
        console.log('📮 [useMailboxSSE] 账户分组事件:', data);
      }
      scheduleMailboxStructureRefresh();
    },
    [scheduleMailboxStructureRefresh]
  );

  const sse = useSSE({
    autoConnect: true,
    onNewEmail: handleNewEmail,
    onEmailStatusChange: handleEmailStatusChange,
    onSyncEvent: handleSyncEvent,
    onNotification: handleNotification,
    onGroupEvent: handleGroupEvent,
    onAccountGroupEvent: handleAccountGroupEvent,
  });

  useEffect(() => {
    return () => {
      if (groupRefreshTimerRef.current) {
        clearTimeout(groupRefreshTimerRef.current);
      }
    };
  }, []);

  // 清除新邮件计数
  const clearNewEmailCount = useCallback(() => {
    setNewEmailCount(0);
  }, []);

  // 获取账户同步状态
  const getAccountSyncStatus = useCallback(
    (accountId: number) => {
      return syncStatus[accountId];
    },
    [syncStatus]
  );

  return {
    ...sse,
    newEmailCount,
    syncStatus,
    clearNewEmailCount,
    getAccountSyncStatus,
  };
}
