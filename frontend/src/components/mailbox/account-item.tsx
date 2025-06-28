'use client';

import { useState } from 'react';
import { ChevronDown, ChevronRight, Circle, AlertCircle, Loader2 } from 'lucide-react';
import { EmailAccount } from '@/types/email';
import { useMailboxStore, useContextMenuStore } from '@/lib/store';
import { FolderTree } from './folder-tree';
import { apiClient } from '@/lib/api';

interface AccountItemProps {
  account: EmailAccount;
}

export function AccountItem({ account }: AccountItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const { selectedAccount, selectAccount, folders, setFolders } = useMailboxStore();
  const { openMenu } = useContextMenuStore();

  // 获取当前账户的文件夹
  const accountFolders = folders.filter((folder) => folder.account_id === account.id);

  // 加载文件夹数据
  const loadFolders = async () => {
    if (accountFolders.length > 0) return; // 已有数据，不重复加载

    setIsLoading(true);
    try {
      const response = await apiClient.getFolders(account.id);
      if (response.success && response.data) {
        setFolders(response.data);
      }
    } catch (error) {
      console.error('Failed to load folders:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // 处理账户点击
  const handleAccountClick = () => {
    setIsExpanded(!isExpanded);
    selectAccount(account);

    if (!isExpanded) {
      loadFolders();
    }
  };

  // 处理右键菜单
  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    openMenu(
      { x: e.clientX, y: e.clientY },
      {
        type: 'account',
        id: account.id,
        data: account,
      }
    );
  };

  // 获取状态指示器
  const getStatusIndicator = () => {
    switch (account.sync_status) {
      case 'syncing':
        return <Loader2 className="w-3 h-3 text-blue-500 animate-spin" />;
      case 'error':
        return <AlertCircle className="w-3 h-3 text-red-500" />;
      case 'success':
        return <Circle className="w-3 h-3 text-green-500 fill-current" />;
      default:
        return <Circle className="w-3 h-3 text-gray-400" />;
    }
  };

  // 获取提供商显示名称
  const getProviderDisplayName = () => {
    const providerNames: Record<string, string> = {
      gmail: 'Gmail',
      outlook: 'Outlook',
      qq: 'QQ邮箱',
      '163': '163邮箱',
      custom: '自定义',
    };
    return providerNames[account.provider] || account.provider;
  };

  return (
    <div className="space-y-1">
      {/* 账户头部 */}
      <div
        onClick={handleAccountClick}
        onContextMenu={handleContextMenu}
        className={`
          flex items-center gap-2 p-2 rounded-md cursor-pointer transition-colors
          hover:bg-gray-100 dark:hover:bg-gray-700
          ${
            selectedAccount?.id === account.id
              ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
              : 'text-gray-700 dark:text-gray-300'
          }
        `}
      >
        {/* 展开/折叠图标 */}
        <div className="flex-shrink-0">
          {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
        </div>

        {/* 状态指示器 */}
        <div className="flex-shrink-0">{getStatusIndicator()}</div>

        {/* 账户信息 */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium truncate">{account.name}</div>
              <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                {getProviderDisplayName()}
              </div>
            </div>

            {/* 未读邮件数量 */}
            {account.unread_emails > 0 && (
              <div className="flex-shrink-0 ml-2">
                <span className="inline-flex items-center justify-center px-2 py-1 text-xs font-medium text-white bg-blue-500 rounded-full min-w-[20px]">
                  {account.unread_emails > 99 ? '99+' : account.unread_emails}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 文件夹列表 */}
      {isExpanded && (
        <div className="ml-6 space-y-1">
          {isLoading ? (
            <div className="flex items-center gap-2 p-2 text-sm text-gray-500 dark:text-gray-400">
              <Loader2 className="w-4 h-4 animate-spin" />
              加载文件夹...
            </div>
          ) : accountFolders.length > 0 ? (
            <FolderTree folders={accountFolders} />
          ) : (
            <div className="p-2 text-sm text-gray-500 dark:text-gray-400">暂无文件夹</div>
          )}
        </div>
      )}

      {/* 错误信息显示 */}
      {account.sync_status === 'error' && account.error_message && (
        <div className="ml-6 p-2 text-xs text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
          {account.error_message}
        </div>
      )}
    </div>
  );
}
