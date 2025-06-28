'use client';

import { useEffect, useCallback, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Mail, Plus, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMailboxStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import {
  MobileLayout,
  MobilePage,
  MobileContent,
  MobileList,
  MobileListItem,
  MobileEmptyState,
  MobileLoading,
} from './mobile-layout';
import { MobileAccountItem } from './mobile-account-item';
import { AccountsHeader } from './mobile-header';
import type { EmailAccount } from '@/types/email';

export function MobileAccountsPage() {
  const { accounts, selectedAccount, selectAccount, isLoading, setAccounts, removeAccount } =
    useMailboxStore();
  const router = useRouter();

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

  // 处理添加邮箱
  const handleAddAccount = () => {
    router.push('/add-email');
  };

  // 处理设置
  const handleSettings = () => {
    // TODO: 实现设置页面
    console.log('打开设置');
  };

  // 处理账户设置
  const handleAccountSettings = (account: EmailAccount) => {
    router.push(`/mailbox/mobile/account/${account.id}/settings`);
  };

  // 处理账户删除
  const handleAccountDelete = async (account: EmailAccount) => {
    const confirmed = confirm(
      `确定要删除账户 "${account.name}" 吗？此操作不可撤销，将删除该账户的所有邮件数据。`
    );
    if (!confirmed) return;

    try {
      const response = await apiClient.deleteEmailAccount(account.id);
      if (response.success) {
        removeAccount(account.id);
        toast.success('账户已删除');
      } else {
        throw new Error(response.message || '删除失败');
      }
    } catch (error) {
      console.error('Delete account failed:', error);
      toast.error(error instanceof Error ? error.message : '删除账户失败');
    }
  };

  // 处理账户点击
  const handleAccountClick = (account: EmailAccount) => {
    selectAccount(account);
    router.push(`/mailbox/mobile/account/${account.id}`);
  };

  if (isLoading) {
    return (
      <MobileLayout>
        <MobilePage>
          <AccountsHeader />
          <MobileContent>
            <MobileLoading message="加载邮箱列表..." />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  return (
    <MobileLayout>
      <MobilePage>
        <AccountsHeader />

        <MobileContent padding={false}>
          {accounts.length > 0 ? (
            <MobileList>
              {accounts.map((account) => (
                <MobileAccountItem
                  key={account.id}
                  account={account}
                  onClick={() => handleAccountClick(account)}
                  onSettings={handleAccountSettings}
                  onDelete={handleAccountDelete}
                  active={selectedAccount?.id === account.id}
                />
              ))}
            </MobileList>
          ) : (
            <MobileEmptyState
              icon={<Mail className="w-8 h-8 text-gray-400" />}
              title="暂无邮箱账户"
              description="添加您的第一个邮箱账户开始使用"
              action={
                <Button onClick={handleAddAccount} className="mt-4">
                  <Plus className="w-4 h-4 mr-2" />
                  添加邮箱
                </Button>
              }
            />
          )}
        </MobileContent>

        {/* 底部操作区域 */}
        {accounts.length > 0 && (
          <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-4">
            <div className="flex gap-3">
              <Button variant="outline" onClick={handleAddAccount} className="flex-1">
                <Plus className="w-4 h-4 mr-2" />
                添加邮箱
              </Button>

              <Button variant="outline" onClick={handleSettings} className="px-4">
                <Settings className="w-4 h-4" />
              </Button>
            </div>
          </div>
        )}
      </MobilePage>
    </MobileLayout>
  );
}
