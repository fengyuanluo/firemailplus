/**
 * 状态管理协调器
 * 处理不同store之间的协调逻辑
 */

import { useMailboxStore, useUIStore } from './store';

// 状态协调器类
export class StoreCoordinator {
  private static instance: StoreCoordinator;

  private constructor() {
    this.setupAccountChangeHandler();
    this.setupFolderChangeHandler();
  }

  public static getInstance(): StoreCoordinator {
    if (!StoreCoordinator.instance) {
      StoreCoordinator.instance = new StoreCoordinator();
    }
    return StoreCoordinator.instance;
  }

  // 处理账户切换时的状态协调
  private setupAccountChangeHandler() {
    let previousAccountId: number | null = null;

    useMailboxStore.subscribe((state) => {
      const currentAccountId = state.selectedAccount?.id || null;

      if (previousAccountId !== currentAccountId) {
        // 账户切换时清理相关状态
        this.handleAccountChange(currentAccountId);
        previousAccountId = currentAccountId;
      }
    });
  }

  // 处理文件夹切换时的状态协调
  private setupFolderChangeHandler() {
    let previousFolderId: number | null = null;

    useMailboxStore.subscribe((state) => {
      const currentFolderId = state.selectedFolder?.id || null;

      if (previousFolderId !== currentFolderId) {
        // 文件夹切换时清理邮件状态
        this.handleFolderChange(state.selectedFolder);
        previousFolderId = currentFolderId;
      }
    });
  }

  // 账户切换处理
  private handleAccountChange(accountId: number | null) {
    const mailboxStore = useMailboxStore.getState();

    // 清理邮件状态
    mailboxStore.setEmails([]);
    mailboxStore.selectEmail(null);
    mailboxStore.clearSelection();
    mailboxStore.setPage(1);

    // 清理文件夹状态
    mailboxStore.setFolders([]);
    mailboxStore.selectFolder(null);

    // 清理搜索状态
    mailboxStore.setSearchQuery('');

    console.log('Account changed to:', accountId);
  }

  // 文件夹切换处理
  private handleFolderChange(folder: any) {
    const mailboxStore = useMailboxStore.getState();

    // 清理邮件选择状态
    mailboxStore.selectEmail(null);
    mailboxStore.clearSelection();
    mailboxStore.setPage(1);

    console.log('Folder changed to:', folder);
  }

  // 执行搜索
  public async performSearch(query: string, filters: any = {}) {
    const mailboxStore = useMailboxStore.getState();

    try {
      mailboxStore.setLoading(true);
      mailboxStore.setSearchQuery(query);

      // TODO: 这里应该调用实际的搜索API
      console.log('Performing search:', query, filters);

      // 模拟搜索结果
      const results = { emails: [], total: 0 };
      mailboxStore.setEmails(results.emails);
    } catch (error) {
      console.error('Search failed:', error);
    } finally {
      mailboxStore.setLoading(false);
    }
  }

  // 批量操作邮件
  public async performBulkOperation(operation: string, emailIds: number[]) {
    const mailboxStore = useMailboxStore.getState();

    try {
      // mailboxStore.setBulkOperating(true); // 方法不存在，注释掉

      // TODO: 这里应该调用实际的批量操作API
      console.log('Performing bulk operation:', operation, emailIds);

      // 根据操作类型更新本地状态
      switch (operation) {
        case 'mark_read':
          emailIds.forEach((id) => {
            mailboxStore.updateEmail(id, { is_read: true });
          });
          break;

        case 'mark_unread':
          emailIds.forEach((id) => {
            mailboxStore.updateEmail(id, { is_read: false });
          });
          break;

        case 'delete':
          emailIds.forEach((id) => {
            mailboxStore.removeEmail(id);
          });
          break;

        case 'archive':
          emailIds.forEach((id) => {
            mailboxStore.removeEmail(id);
          });
          break;
      }

      // 清理选择状态
      mailboxStore.clearSelection();
    } catch (error) {
      console.error('Bulk operation failed:', error);
    } finally {
      // mailboxStore.setBulkOperating(false); // 方法不存在，注释掉
    }
  }

  // 同步邮件
  public async syncEmails(accountId?: number, folderId?: number) {
    const mailboxStore = useMailboxStore.getState();

    try {
      // mailboxStore.setSyncing(true); // 方法不存在，注释掉
      // mailboxStore.setSyncError(null); // 方法不存在，注释掉

      // TODO: 这里应该调用实际的同步API
      console.log('Syncing emails:', accountId, folderId);

      // mailboxStore.updateLastSyncTime(); // 方法不存在，注释掉
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '同步失败';
      // mailboxStore.setSyncError(errorMessage); // 方法不存在，注释掉
      console.error('Sync failed:', errorMessage);
    } finally {
      // mailboxStore.setSyncing(false); // 方法不存在，注释掉
    }
  }

  // 重置所有状态
  public resetAllStates() {
    const mailboxStore = useMailboxStore.getState();

    mailboxStore.setEmails([]);
    mailboxStore.selectEmail(null);
    mailboxStore.clearSelection();
    mailboxStore.setFolders([]);
    mailboxStore.selectFolder(null);
    mailboxStore.resetState();

    console.log('All states reset');
  }
}

// 导出单例实例
export const storeCoordinator = StoreCoordinator.getInstance();

// 导出便捷方法
export const performSearch = storeCoordinator.performSearch.bind(storeCoordinator);
export const performBulkOperation = storeCoordinator.performBulkOperation.bind(storeCoordinator);
export const syncEmails = storeCoordinator.syncEmails.bind(storeCoordinator);
export const resetAllStates = storeCoordinator.resetAllStates.bind(storeCoordinator);
