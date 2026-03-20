import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import type { EmailAccount, EmailGroup, Folder } from '@/types/email';

function resolveApiErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

export async function loadEmailAccountsIntoStore(): Promise<EmailAccount[]> {
  const response = await apiClient.getEmailAccounts();
  if (!response.success || !response.data) {
    throw new Error(response.message || '加载邮箱账户失败');
  }

  useMailboxStore.getState().setAccounts(response.data);
  return response.data;
}

export async function loadEmailGroupsIntoStore(): Promise<EmailGroup[]> {
  const response = await apiClient.getEmailGroups();
  if (!response.success || !response.data) {
    throw new Error(response.message || '加载邮箱分组失败');
  }

  useMailboxStore.getState().setGroups(response.data);
  return response.data;
}

export async function loadFoldersIntoStore(accountId: number): Promise<Folder[]> {
  const response = await apiClient.getFolders(accountId);
  if (!response.success || !response.data) {
    throw new Error(response.message || '加载文件夹失败');
  }

  useMailboxStore.getState().setFolders(response.data);
  return response.data;
}

export async function refreshEmailAccountsAndGroupsIntoStore(): Promise<{
  accounts: EmailAccount[];
  groups: EmailGroup[];
}> {
  try {
    const [accounts, groups] = await Promise.all([
      loadEmailAccountsIntoStore(),
      loadEmailGroupsIntoStore(),
    ]);

    return { accounts, groups };
  } catch (error) {
    throw new Error(resolveApiErrorMessage(error, '刷新邮箱账户与分组失败'));
  }
}

export async function refreshSelectedAccountFoldersIntoStore(): Promise<Folder[] | null> {
  const { selectedAccount, selectedFolder } = useMailboxStore.getState();
  const accountId = selectedAccount?.id ?? selectedFolder?.account_id;

  if (!accountId) {
    return null;
  }

  try {
    return await loadFoldersIntoStore(accountId);
  } catch (error) {
    throw new Error(resolveApiErrorMessage(error, '刷新当前账户文件夹失败'));
  }
}

export async function refreshMailboxSidebarIntoStore(): Promise<{
  accounts: EmailAccount[];
  groups: EmailGroup[];
  folders: Folder[] | null;
}> {
  const { accounts, groups } = await refreshEmailAccountsAndGroupsIntoStore();
  const folders = await refreshSelectedAccountFoldersIntoStore();
  return { accounts, groups, folders };
}
