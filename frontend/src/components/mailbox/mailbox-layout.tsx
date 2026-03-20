'use client';

import { useEffect, useCallback, type ReactNode } from 'react';
import { useUIStore, useMailboxStore, useNotificationStore } from '@/lib/store';
import { useRouteSync, useRouteStatePersist } from '@/hooks/use-route-sync';
import { useResponsive } from '@/hooks/use-responsive';
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts';
import { MailboxSSEBridge, useMailboxSSE } from '@/hooks/use-sse';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { SearchBar } from './search-bar';
import { LeftSidebar } from './left-sidebar';
import { EmailList } from './email-list';
import { EmailDetail } from './email-detail';
import { ComposeModal } from './compose-modal';
import { NotificationCenter } from './notification-center';

interface MailboxLayoutProps {
  header?: ReactNode;
  children?: ReactNode;
  showSidebar?: boolean;
}

export function MailboxLayout({ header, children, showSidebar = true }: MailboxLayoutProps) {
  const { isMobile, isTablet } = useResponsive();
  const { sidebarOpen, sidebarOpenMobile, toggleSidebar, setSidebarOpen } = useUIStore();
  const {
    emails,
    selectedEmail,
    selectEmail,
    accounts,
    folders,
    selectedAccount,
    selectedFolder,
    selectAccount,
    selectFolder,
  } = useMailboxStore();
  const { notifications, removeNotification, markAsRead } = useNotificationStore();

  // 启用键盘快捷键
  useKeyboardShortcuts();

  // 启用路由状态同步
  useRouteSync();
  useRouteStatePersist();

  // 启用 SSE 连接
  const { isConnected, state: sseState, newEmailCount } = useMailboxSSE();

  // 处理邮件选择
  const handleEmailSelect = useCallback(
    (emailId: number) => {
      const email = emails.find((e) => e.id === emailId);
      if (email) {
        selectEmail(email);
      }
    },
    [emails, selectEmail]
  );

  // 响应式侧边栏状态管理
  useEffect(() => {
    if (!showSidebar) return;
    if (!isMobile && !sidebarOpen) {
      // 桌面端自动显示左侧边栏（三段式布局）
      setSidebarOpen(true);
      console.log('🏠 [MailboxLayout] 桌面端自动显示左侧边栏');
    }
    // 移动端不需要自动隐藏，因为有独立的状态管理
  }, [isMobile, sidebarOpen, setSidebarOpen, showSidebar]);

  // 获取当前设备对应的侧边栏状态
  const currentSidebarOpen = isMobile ? sidebarOpenMobile : sidebarOpen;

  // 初始化逻辑：自动选择第一个账户和默认文件夹
  useEffect(() => {
    // 如果有账户但没有选中账户，自动选择第一个账户
    if (accounts.length > 0 && !selectedAccount) {
      const firstAccount = accounts[0];
      selectAccount(firstAccount);
      console.log('🏠 [MailboxLayout] 自动选择第一个账户:', firstAccount.name);
    }
  }, [accounts, selectedAccount, selectAccount]);

  useEffect(() => {
    // 如果有选中账户和文件夹，但没有选中文件夹，自动选择收件箱
    if (selectedAccount && folders.length > 0 && !selectedFolder) {
      const accountFolders = folders.filter((f) => f.account_id === selectedAccount.id);
      const inboxFolder = accountFolders.find((f) => f.type === 'inbox') || accountFolders[0];
      if (inboxFolder) {
        selectFolder(inboxFolder);
        console.log('🏠 [MailboxLayout] 自动选择默认文件夹:', inboxFolder.display_name);
      }
    }
  }, [selectedAccount, folders, selectedFolder, selectFolder]);

  // 调试信息：监控 SSE 连接状态（仅开发环境）
  useEffect(() => {
    if (process.env.NODE_ENV === 'development') {
      console.log('🔗 [MailboxLayout] SSE 状态:', {
        isConnected,
        state: sseState,
        newEmailCount,
        timestamp: new Date().toISOString(),
      });
    }
  }, [isConnected, sseState, newEmailCount]);

  // 移动端重定向逻辑已移至 RouteGuard 组件

  // 如果是移动端，显示加载状态（等待重定向）
  if (isMobile) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  const headerNode = header === undefined ? <SearchBar /> : header;
  const shouldWrapHeader = header === undefined;
  const shouldRenderDefaultContent = children === undefined;

  return (
    <div className="h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <MailboxSSEBridge />
      {/* 顶部搜索框 */}
      {headerNode &&
        (shouldWrapHeader ? (
          <div className="flex-shrink-0 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
            {headerNode}
          </div>
        ) : (
          headerNode
        ))}

      {/* 主要内容区域 */}
      <div className="flex-1 flex overflow-hidden">
        {/* 移动端遮罩层 */}
        {showSidebar && isMobile && currentSidebarOpen && (
          <div
            className="fixed inset-0 bg-black bg-opacity-50 z-40 md:hidden"
            onClick={toggleSidebar}
          />
        )}

        {/* 左侧边栏 */}
        {showSidebar && (
          <div
            className={`
            flex-shrink-0 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700
            ${
              isMobile
                ? `fixed left-0 top-0 h-full w-80 z-50 transform transition-transform duration-300 ${
                    currentSidebarOpen ? 'translate-x-0' : '-translate-x-full'
                  }`
                : `${isTablet ? 'w-72' : 'w-80'} ${currentSidebarOpen ? 'block' : 'hidden'}`
            }
          `}
          >
            <LeftSidebar />
          </div>
        )}

        {/* 主要内容区域 - 三段式布局 */}
        <div className="flex-1 flex overflow-hidden">
          {shouldRenderDefaultContent ? (
            <>
              {/* 中间邮件列表 - 占1.5/5宽度 */}
              <div
                className={`
                bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700
                ${isMobile ? 'w-full' : 'flex-shrink-0'}
              `}
                style={
                  isMobile
                    ? undefined
                    : isTablet
                      ? { flex: '1 1 auto', minWidth: 0, maxWidth: 'none' }
                      : { flex: '1.5', minWidth: '300px', maxWidth: '400px' }
                }
              >
                <EmailList
                  selectedEmailId={selectedEmail?.id}
                  onEmailSelect={handleEmailSelect}
                  showPagination={true}
                />
              </div>

              {/* 右侧邮件详情 - 占3.5/5宽度 */}
              <div
                className={`
                flex-1 bg-white dark:bg-gray-800
                ${isMobile || isTablet ? 'hidden' : 'block'}
              `}
                style={{ flex: '3.5' }}
              >
                <EmailDetail />
              </div>
            </>
          ) : (
            children
          )}
        </div>
      </div>

      {isTablet && (
        <Sheet open={!!selectedEmail} onOpenChange={(open) => !open && selectEmail(null)}>
          <SheetContent side="right" className="w-full sm:max-w-2xl p-0">
            <SheetHeader className="sr-only">
              <SheetTitle>邮件详情</SheetTitle>
            </SheetHeader>
            <div className="h-full bg-white dark:bg-gray-800">
              <EmailDetail />
            </div>
          </SheetContent>
        </Sheet>
      )}

      {/* 写信弹窗 */}
      <ComposeModal />

      <NotificationCenter
        notifications={notifications}
        onRemove={removeNotification}
        onMarkAsRead={markAsRead}
      />
    </div>
  );
}
