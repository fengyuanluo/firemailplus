/**
 * 路由状态同步Hook
 * 确保页面状态与路由保持一致，防止状态丢失
 */

import { useEffect } from 'react';
import { usePathname, useSearchParams } from 'next/navigation';
import { useMailboxStore } from '@/lib/store';

export function useRouteSync() {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const {
    selectedAccount,
    selectedFolder,
    selectedEmail,
    searchQuery,
    selectAccount,
    selectFolder,
    selectEmail,
    setSearchQuery,
    accounts,
    folders,
    emails,
  } = useMailboxStore();

  // 同步搜索状态
  useEffect(() => {
    const urlSearchQuery = searchParams.get('q') || '';

    // 如果URL中的搜索查询与store中的不一致，更新store
    if (urlSearchQuery !== searchQuery) {
      setSearchQuery(urlSearchQuery);
    }
  }, [searchParams, searchQuery, setSearchQuery]);

  // 同步账户状态
  useEffect(() => {
    // 从路径中解析账户ID
    const accountMatch = pathname.match(/\/account\/(\d+)/);
    const accountId = accountMatch ? parseInt(accountMatch[1]) : null;

    if (accountId && accounts.length > 0) {
      const account = accounts.find((acc) => acc.id === accountId);
      if (account && selectedAccount?.id !== accountId) {
        selectAccount(account);
      }
    }
  }, [pathname, accounts, selectedAccount, selectAccount]);

  // 同步文件夹状态
  useEffect(() => {
    // 从路径中解析文件夹ID
    const folderMatch = pathname.match(/\/folder\/(\d+)/);
    const folderId = folderMatch ? parseInt(folderMatch[1]) : null;

    if (folderId && folders.length > 0) {
      const folder = folders.find((f) => f.id === folderId);
      if (folder && selectedFolder?.id !== folderId) {
        selectFolder(folder);
      }
    }
  }, [pathname, folders, selectedFolder, selectFolder]);

  // 同步邮件状态
  useEffect(() => {
    // 从路径中解析邮件ID
    const emailMatch = pathname.match(/\/email\/(\d+)/);
    const emailId = emailMatch ? parseInt(emailMatch[1]) : null;

    if (emailId && emails.length > 0) {
      const email = emails.find((e) => e.id === emailId);
      if (email && selectedEmail?.id !== emailId) {
        selectEmail(email);
      }
    } else if (!emailId && selectedEmail) {
      // 如果路径中没有邮件ID但store中有选中的邮件，清除选中状态
      selectEmail(null);
    }
  }, [pathname, emails, selectedEmail, selectEmail]);

  // 处理页面刷新时的状态恢复
  useEffect(() => {
    // 如果是邮件详情页面但没有选中的邮件，尝试从URL恢复
    const emailMatch = pathname.match(/\/email\/(\d+)/);
    if (emailMatch && !selectedEmail) {
      const emailId = parseInt(emailMatch[1]);
      // 这里可以触发邮件详情的加载
      // 实际实现中应该调用API获取邮件详情
      console.log('需要加载邮件详情:', emailId);
    }

    // 如果是文件夹页面但没有选中的文件夹，尝试从URL恢复
    const folderMatch = pathname.match(/\/folder\/(\d+)/);
    if (folderMatch && !selectedFolder) {
      const folderId = parseInt(folderMatch[1]);
      // 这里可以触发文件夹信息的加载
      console.log('需要加载文件夹信息:', folderId);
    }

    // 如果是账户页面但没有选中的账户，尝试从URL恢复
    const accountMatch = pathname.match(/\/account\/(\d+)/);
    if (accountMatch && !selectedAccount) {
      const accountId = parseInt(accountMatch[1]);
      // 这里可以触发账户信息的加载
      console.log('需要加载账户信息:', accountId);
    }
  }, [pathname, selectedEmail, selectedFolder, selectedAccount]);

  return {
    // 返回当前路由解析的状态
    routeState: {
      accountId: pathname.match(/\/account\/(\d+)/)?.[1],
      folderId: pathname.match(/\/folder\/(\d+)/)?.[1],
      emailId: pathname.match(/\/email\/(\d+)/)?.[1],
      searchQuery: searchParams.get('q') || '',
      isSearchPage: pathname.includes('/search'),
      isMobilePage: pathname.includes('/mobile'),
    },
  };
}

// 路由状态恢复Hook
export function useRouteStateRestore() {
  const { routeState } = useRouteSync();
  const { accounts, folders, emails, setAccounts, setFolders, setEmails } = useMailboxStore();

  // 根据路由状态恢复数据
  useEffect(() => {
    const restoreState = async () => {
      try {
        // 如果有账户ID但没有账户数据，加载账户列表
        if (routeState.accountId && accounts.length === 0) {
          // 这里应该调用API加载账户列表
          console.log('需要加载账户列表');
        }

        // 如果有文件夹ID但没有文件夹数据，加载文件夹列表
        if (routeState.folderId && folders.length === 0) {
          // 这里应该调用API加载文件夹列表
          console.log('需要加载文件夹列表');
        }

        // 如果有邮件ID但没有邮件数据，加载邮件列表
        if (routeState.emailId && emails.length === 0) {
          // 这里应该调用API加载邮件列表
          console.log('需要加载邮件列表');
        }
      } catch (error) {
        console.error('路由状态恢复失败:', error);
      }
    };

    restoreState();
  }, [routeState, accounts, folders, emails, setAccounts, setFolders, setEmails]);
}

// 路由状态持久化Hook
export function useRouteStatePersist() {
  const { selectedAccount, selectedFolder, selectedEmail, searchQuery } = useMailboxStore();

  // 将重要状态保存到sessionStorage
  useEffect(() => {
    const stateToSave = {
      selectedAccountId: selectedAccount?.id,
      selectedFolderId: selectedFolder?.id,
      selectedEmailId: selectedEmail?.id,
      searchQuery,
      timestamp: Date.now(),
    };

    try {
      sessionStorage.setItem('mailbox_route_state', JSON.stringify(stateToSave));
    } catch (error) {
      console.warn('无法保存路由状态到sessionStorage:', error);
    }
  }, [selectedAccount, selectedFolder, selectedEmail, searchQuery]);

  // 从sessionStorage恢复状态
  useEffect(() => {
    try {
      const savedState = sessionStorage.getItem('mailbox_route_state');
      if (savedState) {
        const parsed = JSON.parse(savedState);

        // 检查状态是否过期（1小时）
        const isExpired = Date.now() - parsed.timestamp > 60 * 60 * 1000;
        if (isExpired) {
          sessionStorage.removeItem('mailbox_route_state');
          return;
        }

        // 这里可以根据保存的状态恢复选中项
        console.log('从sessionStorage恢复状态:', parsed);
      }
    } catch (error) {
      console.warn('无法从sessionStorage恢复路由状态:', error);
    }
  }, []);
}
