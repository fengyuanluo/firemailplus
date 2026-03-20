'use client';

import { useCallback } from 'react';
import { toast } from 'sonner';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import {
  loadEmailAccountsIntoStore,
  loadEmailGroupsIntoStore,
  refreshEmailAccountsAndGroupsIntoStore,
} from '@/lib/mailbox-group-data';
import type { EmailAccount, EmailGroup, Email } from '@/types/email';

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

export function useEmailGroupActions() {
  const accounts = useMailboxStore((state) => state.accounts);
  const folders = useMailboxStore((state) => state.folders);
  const emails = useMailboxStore((state) => state.emails);
  const selectedAccount = useMailboxStore((state) => state.selectedAccount);
  const setFolders = useMailboxStore((state) => state.setFolders);
  const setEmails = useMailboxStore((state) => state.setEmails);
  const updateAccount = useMailboxStore((state) => state.updateAccount);
  const clearAccountSelection = useMailboxStore((state) => state.clearAccountSelection);

  const loadAccounts = useCallback(async () => {
    return loadEmailAccountsIntoStore();
  }, []);

  const loadGroups = useCallback(async () => {
    return loadEmailGroupsIntoStore();
  }, []);

  const refreshAccountsAndGroups = useCallback(async () => {
    return refreshEmailAccountsAndGroupsIntoStore();
  }, []);

  const createGroup = useCallback(
    async (name: string) => {
      const response = await apiClient.createEmailGroup({ name: name.trim() });
      if (!response.success) {
        throw new Error(response.message || '创建分组失败');
      }
      await refreshAccountsAndGroups();
      toast.success('分组已创建');
      return response.data;
    },
    [refreshAccountsAndGroups]
  );

  const renameGroup = useCallback(
    async (group: EmailGroup, name: string) => {
      const response = await apiClient.updateEmailGroup(group.id, { name: name.trim() });
      if (!response.success) {
        throw new Error(response.message || '更新分组失败');
      }
      await loadGroups();
      toast.success('分组已更新');
      return response.data;
    },
    [loadGroups]
  );

  const deleteGroup = useCallback(
    async (group: EmailGroup, defaultGroupName: string) => {
      const response = await apiClient.deleteEmailGroup(group.id);
      if (!response.success) {
        throw new Error(response.message || '删除分组失败');
      }
      await refreshAccountsAndGroups();
      toast.success(`分组已删除，相关邮箱已移动到默认分组“${defaultGroupName}”`);
    },
    [refreshAccountsAndGroups]
  );

  const setDefaultGroup = useCallback(
    async (group: EmailGroup) => {
      const response = await apiClient.setDefaultEmailGroup(group.id);
      if (!response.success) {
        throw new Error(response.message || '设置默认分组失败');
      }
      await refreshAccountsAndGroups();
      toast.success(`已将“${group.name}”设为默认分组`);
      return response.data;
    },
    [refreshAccountsAndGroups]
  );

  const moveAccountToGroup = useCallback(
    async (accountId: number, targetGroupId: number | null) => {
      const response = await apiClient.updateEmailAccount(accountId, {
        group_id: targetGroupId,
      });
      if (!response.success) {
        throw new Error(response.message || '移动邮箱到分组失败');
      }
      await refreshAccountsAndGroups();
      toast.success('邮箱分组已更新');
      return response.data;
    },
    [refreshAccountsAndGroups]
  );

  const persistGroupOrder = useCallback(
    async (groupIds: number[]) => {
      const response = await apiClient.reorderEmailGroups(groupIds);
      if (!response.success) {
        throw new Error(response.message || '分组排序保存失败');
      }
      await loadGroups();
      toast.success('分组排序已保存');
      return response.data;
    },
    [loadGroups]
  );

  const deleteAccount = useCallback(
    async (account: EmailAccount) => {
      const response = await apiClient.deleteEmailAccount(account.id);
      if (!response.success) {
        throw new Error(response.message || '删除账户失败');
      }
      await refreshAccountsAndGroups();
      toast.success('账户已删除');
    },
    [refreshAccountsAndGroups]
  );

  const batchSyncAccounts = useCallback(async (accountIds: number[]) => {
    const response = await apiClient.batchSyncEmailAccounts(accountIds);
    if (!response.success) {
      throw new Error(response.message || '批量同步失败');
    }
    toast.success('已开始批量同步');
  }, []);

  const batchMarkAccountsRead = useCallback(
    async (accountIds: number[]) => {
      const response = await apiClient.batchMarkAccountsAsRead(accountIds);
      if (!response.success) {
        throw new Error(response.message || '批量标记已读失败');
      }

      toast.success(`已标记 ${accountIds.length} 个邮箱的邮件为已读`);

      accountIds.forEach((id) => {
        const account = accounts.find((item) => item.id === id);
        if (account) {
          updateAccount({ ...account, unread_emails: 0 });
        }
      });

      if (folders.length > 0) {
        setFolders(
          folders.map((folder) =>
            accountIds.includes(folder.account_id) ? { ...folder, unread_emails: 0 } : folder
          )
        );
      }

      if (emails.length > 0 && selectedAccount && accountIds.includes(selectedAccount.id)) {
        const updatedEmails: Email[] = emails.map((email) => ({ ...email, is_read: true }));
        setEmails(updatedEmails);
      }
    },
    [accounts, emails, folders, selectedAccount, setEmails, setFolders, updateAccount]
  );

  const batchDeleteAccounts = useCallback(
    async (accountIds: number[]) => {
      const response = await apiClient.batchDeleteEmailAccounts(accountIds);
      if (!response.success) {
        throw new Error(response.message || '批量删除失败');
      }
      await refreshAccountsAndGroups();
      clearAccountSelection();
      toast.success('已删除所选邮箱');
    },
    [clearAccountSelection, refreshAccountsAndGroups]
  );

  return {
    loadAccounts,
    loadGroups,
    refreshAccountsAndGroups,
    createGroup,
    renameGroup,
    deleteGroup,
    setDefaultGroup,
    moveAccountToGroup,
    persistGroupOrder,
    deleteAccount,
    batchSyncAccounts,
    batchMarkAccountsRead,
    batchDeleteAccounts,
    getErrorMessage,
  };
}
