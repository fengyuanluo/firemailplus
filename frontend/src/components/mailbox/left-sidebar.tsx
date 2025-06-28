'use client';

import { useEffect, useCallback, useState } from 'react';
import { useMailboxStore } from '@/lib/store';
import { SidebarHeader } from './sidebar-header';
import { AccountItem } from './account-item';
import { ContextMenu } from './context-menu';
import { AccountSettingsModal } from './account-settings-modal';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import type { EmailAccount } from '@/types/email';

export function LeftSidebar() {
  const { accounts, setAccounts, removeAccount } = useMailboxStore();
  const [settingsAccount, setSettingsAccount] = useState<EmailAccount | null>(null);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  // 加载邮箱账户列表 - 使用useCallback避免重复请求
  const loadAccounts = useCallback(async () => {
    try {
      const response = await apiClient.getEmailAccounts();
      if (response.success && response.data) {
        setAccounts(response.data);
      }
    } catch (error) {
      console.error('Failed to load accounts:', error);
    }
  }, [setAccounts]);

  useEffect(() => {
    // 只在账户列表为空时加载
    if (accounts.length === 0) {
      loadAccounts();
    }
  }, [accounts.length, loadAccounts]);

  // 处理右键菜单操作
  const handleMarkAllAsRead = (targetId: number) => {
    // TODO: 更新本地状态
    console.log('Mark all as read:', targetId);
  };

  const handleSync = (targetId: number) => {
    // TODO: 更新同步状态
    console.log('Sync:', targetId);
  };

  const handleDelete = (targetId: number) => {
    // TODO: 从本地状态中移除
    console.log('Delete:', targetId);
  };

  const handleRename = (targetId: number) => {
    // TODO: 显示重命名对话框
    console.log('Rename:', targetId);
  };

  const handleSettings = (targetId: number) => {
    const account = accounts.find((acc) => acc.id === targetId);
    if (account) {
      setSettingsAccount(account);
      setIsSettingsOpen(true);
    }
  };

  const handleCloseSettings = () => {
    setIsSettingsOpen(false);
    setSettingsAccount(null);
  };

  const handleDeleteAccount = async (targetId: number) => {
    const account = accounts.find((acc) => acc.id === targetId);
    if (!account) return;

    const confirmed = confirm(
      `确定要删除账户 "${account.name}" 吗？此操作不可撤销，将删除该账户的所有邮件数据。`
    );
    if (!confirmed) return;

    try {
      const response = await apiClient.deleteEmailAccount(targetId);
      if (response.success) {
        removeAccount(targetId);
        toast.success('账户已删除');
      } else {
        throw new Error(response.message || '删除失败');
      }
    } catch (error: any) {
      console.error('Delete account failed:', error);
      toast.error(error.message || '删除账户失败');
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* 顶部固定按钮区域 */}
      <SidebarHeader />

      {/* 邮箱账户列表区域 */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4">
          {accounts.length > 0 ? (
            <div className="space-y-2">
              <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-3">
                邮箱账户
              </div>

              <div className="space-y-2">
                {accounts.map((account) => (
                  <AccountItem key={account.id} account={account} />
                ))}
              </div>
            </div>
          ) : (
            <div className="text-center py-8">
              <div className="text-gray-500 dark:text-gray-400 text-sm">暂无邮箱账户</div>
              <div className="text-xs text-gray-400 dark:text-gray-500 mt-1">请先添加邮箱账户</div>
            </div>
          )}
        </div>
      </div>

      {/* 右键菜单 */}
      <ContextMenu
        onMarkAllAsRead={handleMarkAllAsRead}
        onSync={handleSync}
        onDelete={handleDelete}
        onRename={handleRename}
        onSettings={handleSettings}
        onDeleteAccount={handleDeleteAccount}
      />

      {/* 账户设置弹窗 */}
      <AccountSettingsModal
        isOpen={isSettingsOpen}
        onClose={handleCloseSettings}
        account={settingsAccount}
      />
    </div>
  );
}
