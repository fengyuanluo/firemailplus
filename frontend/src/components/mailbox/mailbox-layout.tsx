'use client';

import { useEffect, useRef, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useUIStore, useMailboxStore } from '@/lib/store';
import { useRouteSync, useRouteStatePersist } from '@/hooks/use-route-sync';
import { useIsMobile } from '@/hooks/use-responsive';
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts';
import { useMailboxSSE } from '@/hooks/use-sse';
import { SearchBar } from './search-bar';
import { LeftSidebar } from './left-sidebar';
import { EmailList } from './email-list';
import { EmailDetail } from './email-detail';
import { ComposeModal } from './compose-modal';
import { NotificationCenter } from './notification-center';

export function MailboxLayout() {
  const isMobile = useIsMobile(); // 使用统一的响应式Hook
  const { sidebarOpen, sidebarOpenMobile, toggleSidebar, setSidebarOpen, setSidebarOpenMobile } =
    useUIStore();
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

  const router = useRouter();

  // 启用键盘快捷键
  useKeyboardShortcuts();

  // 启用路由状态同步
  useRouteSync();
  useRouteStatePersist();

  // 启用 SSE 连接
  const { isConnected, state: sseState, newEmailCount, clearNewEmailCount } = useMailboxSSE();

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
    if (!isMobile && !sidebarOpen) {
      // 桌面端自动显示左侧边栏（三段式布局）
      setSidebarOpen(true);
      console.log('🏠 [MailboxLayout] 桌面端自动显示左侧边栏');
    }
    // 移动端不需要自动隐藏，因为有独立的状态管理
  }, [isMobile, sidebarOpen, setSidebarOpen]);

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

  return (
    <div className="h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      {/* 顶部搜索框 */}
      <div className="flex-shrink-0 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <SearchBar />
      </div>

      {/* 主要内容区域 */}
      <div className="flex-1 flex overflow-hidden">
        {/* 移动端遮罩层 */}
        {isMobile && currentSidebarOpen && (
          <div
            className="fixed inset-0 bg-black bg-opacity-50 z-40 md:hidden"
            onClick={toggleSidebar}
          />
        )}

        {/* 左侧边栏 */}
        <div
          className={`
          flex-shrink-0 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700
          ${
            isMobile
              ? `fixed left-0 top-0 h-full w-80 z-50 transform transition-transform duration-300 ${
                  currentSidebarOpen ? 'translate-x-0' : '-translate-x-full'
                }`
              : `w-80 ${currentSidebarOpen ? 'block' : 'hidden'}`
          }
        `}
        >
          <LeftSidebar />
        </div>

        {/* 主要内容区域 - 三段式布局 */}
        <div className="flex-1 flex overflow-hidden">
          {/* 中间邮件列表 - 占1.5/5宽度 */}
          <div
            className={`
            bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700
            ${isMobile ? 'w-full' : 'flex-shrink-0'}
          `}
            style={{ flex: isMobile ? 'none' : '1.5', minWidth: '300px', maxWidth: '400px' }}
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
            ${isMobile ? 'hidden' : 'block'}
          `}
            style={{ flex: '3.5' }}
          >
            <EmailDetail />
          </div>
        </div>
      </div>

      {/* 写信弹窗 */}
      <ComposeModal />

      {/* 通知中心 - 暂时简化 */}
      <NotificationCenter notifications={[]} onRemove={() => {}} onMarkAsRead={() => {}} />
    </div>
  );
}
