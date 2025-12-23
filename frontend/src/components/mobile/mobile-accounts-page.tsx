'use client';

'use client';

import { useEffect, useCallback, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Mail, Plus, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMailboxStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { EmailGroupCard } from '../mailbox/email-group-card';
import {
  MobileLayout,
  MobilePage,
  MobileContent,
  MobileList,
  MobileEmptyState,
  MobileLoading,
} from './mobile-layout';
import { MobileAccountItem } from './mobile-account-item';
import { AccountsHeader } from './mobile-header';
import type { EmailAccount, EmailGroup } from '@/types/email';

export function MobileAccountsPage() {
  const {
    accounts,
    selectedAccount,
    selectAccount,
    isLoading,
    setAccounts,
    removeAccount,
    groups,
    setGroups,
  } = useMailboxStore();
  const router = useRouter();
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<number>>(new Set());

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

  const loadGroups = useCallback(async () => {
    setLoadingGroups(true);
    try {
      const response = await apiClient.getEmailGroups();
      if (response.success && response.data) {
        setGroups(response.data);
      }
    } catch (error) {
      console.error('Failed to load groups:', error);
    } finally {
      setLoadingGroups(false);
    }
  }, [setGroups]);

  useEffect(() => {
    // 只在账户列表为空时加载
    if (accounts.length === 0) {
      loadAccounts();
    }
  }, [accounts.length, loadAccounts]);

  useEffect(() => {
    if (groups.length === 0) {
      loadGroups();
    }
  }, [groups.length, loadGroups]);

  const defaultGroup = useMemo(() => groups.find((g) => g.is_default), [groups]);

  const orderedGroups = useMemo(() => {
    const sorted = [...groups].sort((a, b) => a.sort_order - b.sort_order);
    if (!defaultGroup) return sorted;
    const others = sorted.filter((g) => !g.is_default);
    return [defaultGroup, ...others];
  }, [groups, defaultGroup]);

  const accountsByGroup = useMemo(() => {
    const map = new Map<number, EmailAccount[]>();
    const fallbackId = defaultGroup?.id ?? -1;
    accounts.forEach((account) => {
      const gid = account.group_id ?? fallbackId;
      const list = map.get(gid) ?? [];
      list.push(account);
      map.set(gid, list);
    });
    return map;
  }, [accounts, defaultGroup]);

  const toggleCollapse = (groupId: number) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
      }
      return next;
    });
  };

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

  const hasGroups = orderedGroups.length > 0;

  if (isLoading || loadingGroups) {
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
            hasGroups ? (
              <div className="p-4 space-y-3">
                {orderedGroups.map((group: EmailGroup) => {
                  const accountsInGroup = accountsByGroup.get(group.id) || [];
                  return (
                    <EmailGroupCard
                      key={group.id}
                      group={group}
                      accountsCount={accountsInGroup.length}
                      collapsed={collapsedGroups.has(group.id)}
                      onToggleCollapse={() => toggleCollapse(group.id)}
                      showHandle={false}
                      draggable={false}
                      subtitleSuffix={group.is_default ? '默认' : ''}
                      className="bg-white dark:bg-gray-800 shadow-sm"
                    >
                      {accountsInGroup.length === 0 ? (
                        <div className="text-xs text-gray-500 dark:text-gray-400 border border-dashed border-gray-300 dark:border-gray-700 rounded-md p-3">
                          暂无邮箱，添加或移动邮箱到此分组
                        </div>
                      ) : (
                        accountsInGroup.map((account) => (
                          <MobileAccountItem
                            key={account.id}
                            account={account}
                            onClick={() => handleAccountClick(account)}
                            onSettings={handleAccountSettings}
                            onDelete={handleAccountDelete}
                            active={selectedAccount?.id === account.id}
                          />
                        ))
                      )}
                    </EmailGroupCard>
                  );
                })}
              </div>
            ) : (
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
            )
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
