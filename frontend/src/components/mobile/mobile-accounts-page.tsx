'use client';

import { useEffect, useCallback, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Mail,
  Plus,
  Settings,
  MoreVertical,
  Star,
  Pencil,
  Trash2,
  ArrowUp,
  ArrowDown,
  RefreshCw,
  CheckSquare,
  X,
  FolderPlus,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import { useMailboxStore } from '@/lib/store';
import { EmailGroupCard } from '../mailbox/email-group-card';
import { EmailGroupDialog } from '../mailbox/email-group-dialog';
import {
  MobileLayout,
  MobilePage,
  MobileContent,
  MobileList,
  MobileEmptyState,
  MobileLoading,
} from './mobile-layout';
import { MobileAccountItem } from './mobile-account-item';
import { MobileHeader } from './mobile-header';
import { useEmailGroupActions } from '@/hooks/use-email-group-actions';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import {
  canDeleteEmailGroup,
  canEditEmailGroup,
  canReorderEmailGroup,
  canSetDefaultEmailGroup,
  isHiddenSystemEmailGroup,
  type EmailAccount,
  type EmailGroup,
} from '@/types/email';

export function MobileAccountsPage() {
  const {
    accounts,
    selectedAccount,
    selectAccount,
    isLoading,
    groups,
    selectionMode,
    setSelectionMode,
    selectedAccountIds,
    toggleSelectAccount,
    clearAccountSelection,
    setSelectedAccountIds,
  } = useMailboxStore();
  const router = useRouter();
  const {
    loadAccounts,
    loadGroups,
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
  } = useEmailGroupActions();

  const [loadingData, setLoadingData] = useState(false);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<number>>(new Set());
  const [groupDialogOpen, setGroupDialogOpen] = useState(false);
  const [groupDialogMode, setGroupDialogMode] = useState<'create' | 'edit'>('create');
  const [editingGroup, setEditingGroup] = useState<EmailGroup | null>(null);
  const [moveSheetOpen, setMoveSheetOpen] = useState(false);
  const [movingAccount, setMovingAccount] = useState<EmailAccount | null>(null);

  const loadInitialData = useCallback(async () => {
    setLoadingData(true);
    try {
      await Promise.all([loadAccounts(), loadGroups()]);
    } catch (error) {
      console.error('Failed to load mobile mailbox data:', error);
    } finally {
      setLoadingData(false);
    }
  }, [loadAccounts, loadGroups]);

  useEffect(() => {
    if (accounts.length === 0 || groups.length === 0) {
      void loadInitialData();
    }
  }, [accounts.length, groups.length, loadInitialData]);

  const defaultGroup = useMemo(() => groups.find((group) => group.is_default), [groups]);
  const defaultGroupName = defaultGroup?.name ?? '默认分组';

  const orderedGroups = useMemo(() => {
    const visibleGroups = [...groups]
      .filter((group) => !isHiddenSystemEmailGroup(group))
      .sort((a, b) => a.sort_order - b.sort_order);
    if (!defaultGroup) return visibleGroups;
    const others = visibleGroups.filter((group) => !group.is_default);
    return [defaultGroup, ...others];
  }, [defaultGroup, groups]);

  const accountsByGroup = useMemo(() => {
    const map = new Map<number, EmailAccount[]>();
    const fallbackId = defaultGroup?.id ?? -1;
    accounts.forEach((account) => {
      const groupId = account.group_id ?? fallbackId;
      const list = map.get(groupId) ?? [];
      list.push(account);
      map.set(groupId, list);
    });
    return map;
  }, [accounts, defaultGroup]);

  const reorderableGroups = useMemo(
    () => orderedGroups.filter((group) => canReorderEmailGroup(group)),
    [orderedGroups]
  );

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

  const handleAddAccount = () => {
    router.push('/add-email');
  };

  const handleSettings = () => {
    console.log('打开设置');
  };

  const handleAccountSettings = (account: EmailAccount) => {
    router.push(`/mailbox/mobile/account/${account.id}/settings`);
  };

  const handleAccountDelete = async (account: EmailAccount) => {
    const confirmed = confirm(
      `确定要删除账户 "${account.name}" 吗？此操作不可撤销，将删除该账户的所有邮件数据。`
    );
    if (!confirmed) return;

    try {
      await deleteAccount(account);
    } catch (error) {
      console.error('Delete account failed:', error);
      toast.error(getErrorMessage(error, '删除账户失败'));
    }
  };

  const handleAccountClick = (account: EmailAccount) => {
    if (selectionMode) {
      toggleSelectAccount(account.id);
      return;
    }
    selectAccount(account);
    router.push(`/mailbox/mobile/account/${account.id}`);
  };

  const openCreateGroupDialog = () => {
    setGroupDialogMode('create');
    setEditingGroup(null);
    setGroupDialogOpen(true);
  };

  const openEditGroupDialog = (group: EmailGroup) => {
    setGroupDialogMode('edit');
    setEditingGroup(group);
    setGroupDialogOpen(true);
  };

  const handleSubmitGroupDialog = async (name: string) => {
    try {
      if (groupDialogMode === 'create') {
        await createGroup(name);
      } else if (editingGroup) {
        await renameGroup(editingGroup, name);
      }
      setEditingGroup(null);
    } catch (error) {
      console.error('Save group failed:', error);
      toast.error(getErrorMessage(error, '分组保存失败'));
      throw error;
    }
  };

  const handleDeleteGroup = async (group: EmailGroup) => {
    if (!canDeleteEmailGroup(group)) {
      toast.error(group.is_default ? '默认分组不可删除' : '系统分组不可删除');
      return;
    }

    const confirmed = confirm(
      `删除分组 "${group.name}" 后，其中邮箱将移动到默认分组“${defaultGroupName}”，确认删除？`
    );
    if (!confirmed) return;

    try {
      await deleteGroup(group, defaultGroupName);
    } catch (error) {
      console.error('Delete group failed:', error);
      toast.error(getErrorMessage(error, '删除分组失败'));
    }
  };

  const handleSetDefaultGroup = async (group: EmailGroup) => {
    if (!canSetDefaultEmailGroup(group)) return;

    const confirmed = confirm(`是否将分组“${group.name}”设为默认分组？`);
    if (!confirmed) return;

    try {
      await setDefaultGroup(group);
    } catch (error) {
      console.error('Set default group failed:', error);
      toast.error(getErrorMessage(error, '设置默认分组失败'));
    }
  };

  const moveGroupPosition = async (group: EmailGroup, delta: -1 | 1) => {
    const currentIndex = reorderableGroups.findIndex((item) => item.id === group.id);
    if (currentIndex < 0) return;

    const nextIndex = currentIndex + delta;
    if (nextIndex < 0 || nextIndex >= reorderableGroups.length) return;

    const updated = [...reorderableGroups];
    const [moved] = updated.splice(currentIndex, 1);
    updated.splice(nextIndex, 0, moved);

    try {
      await persistGroupOrder(updated.map((item) => item.id));
    } catch (error) {
      console.error('Move group position failed:', error);
      toast.error(getErrorMessage(error, '分组排序保存失败'));
    }
  };

  const handleOpenMoveGroupSheet = (account: EmailAccount) => {
    setMovingAccount(account);
    setMoveSheetOpen(true);
  };

  const handleMoveAccountConfirm = async (group: EmailGroup) => {
    if (!movingAccount) return;

    try {
      await moveAccountToGroup(movingAccount.id, group.is_default ? null : group.id);
      setMoveSheetOpen(false);
      setMovingAccount(null);
    } catch (error) {
      console.error('Move account group failed:', error);
      toast.error(getErrorMessage(error, '移动邮箱到分组失败'));
    }
  };

  const handleToggleGroupSelection = (group: EmailGroup) => {
    const accountsInGroup = accountsByGroup.get(group.id) || [];
    if (accountsInGroup.length === 0) return;

    const ids = accountsInGroup.map((account) => account.id);
    const allSelected = ids.every((id) => selectedAccountIds.has(id));
    const next = new Set(selectedAccountIds);

    if (allSelected) {
      ids.forEach((id) => next.delete(id));
    } else {
      ids.forEach((id) => next.add(id));
    }

    setSelectedAccountIds(Array.from(next));
  };

  const handleBatchSync = async () => {
    const ids = Array.from(selectedAccountIds);
    if (ids.length === 0) {
      toast.error('请先选择要同步的邮箱');
      return;
    }

    try {
      await batchSyncAccounts(ids);
    } catch (error) {
      toast.error(getErrorMessage(error, '批量同步失败'));
    }
  };

  const handleBatchMarkRead = async () => {
    const ids = Array.from(selectedAccountIds);
    if (ids.length === 0) {
      toast.error('请先选择要标记已读的邮箱');
      return;
    }

    try {
      await batchMarkAccountsRead(ids);
    } catch (error) {
      toast.error(getErrorMessage(error, '批量标记已读失败'));
    }
  };

  const handleBatchDelete = async () => {
    const ids = Array.from(selectedAccountIds);
    if (ids.length === 0) {
      toast.error('请先选择要删除的邮箱');
      return;
    }

    const confirmed = confirm(`确定删除选中的 ${ids.length} 个邮箱账户吗？操作不可撤销。`);
    if (!confirmed) return;

    try {
      await batchDeleteAccounts(ids);
    } catch (error) {
      toast.error(getErrorMessage(error, '批量删除失败'));
    }
  };

  const toggleSelectionMode = () => {
    if (selectionMode) {
      setSelectionMode(false);
      clearAccountSelection();
      return;
    }
    setSelectionMode(true);
  };

  const renderGroupActions = (group: EmailGroup) => {
    const canMoveUp = reorderableGroups.findIndex((item) => item.id === group.id) > 0;
    const canMoveDown =
      reorderableGroups.findIndex((item) => item.id === group.id) >= 0 &&
      reorderableGroups.findIndex((item) => item.id === group.id) < reorderableGroups.length - 1;

    const hasActions =
      canSetDefaultEmailGroup(group) ||
      canEditEmailGroup(group) ||
      canDeleteEmailGroup(group) ||
      canMoveUp ||
      canMoveDown;

    if (!hasActions) return null;

    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={(e) => e.stopPropagation()}>
            <MoreVertical className="w-4 h-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {canSetDefaultEmailGroup(group) && (
            <DropdownMenuItem onClick={() => handleSetDefaultGroup(group)}>
              <Star className="w-4 h-4 mr-2" />
              设为默认
            </DropdownMenuItem>
          )}
          {canEditEmailGroup(group) && (
            <DropdownMenuItem onClick={() => openEditGroupDialog(group)}>
              <Pencil className="w-4 h-4 mr-2" />
              重命名分组
            </DropdownMenuItem>
          )}
          {canMoveUp && (
            <DropdownMenuItem onClick={() => moveGroupPosition(group, -1)}>
              <ArrowUp className="w-4 h-4 mr-2" />
              上移
            </DropdownMenuItem>
          )}
          {canMoveDown && (
            <DropdownMenuItem onClick={() => moveGroupPosition(group, 1)}>
              <ArrowDown className="w-4 h-4 mr-2" />
              下移
            </DropdownMenuItem>
          )}
          {canDeleteEmailGroup(group) && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => handleDeleteGroup(group)} className="text-red-600">
                <Trash2 className="w-4 h-4 mr-2" />
                删除分组
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    );
  };

  if (isLoading || loadingData) {
    return (
      <MobileLayout>
        <MobilePage>
          <MobileHeader title="邮箱" />
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
        <MobileHeader
          title="邮箱"
          showSearch={true}
          showCompose={true}
          rightContent={
            <div className="flex items-center gap-1">
              <Button variant={selectionMode ? 'default' : 'ghost'} size="sm" onClick={toggleSelectionMode}>
                {selectionMode ? '退出' : '选择'}
              </Button>
              <Button variant="ghost" size="sm" onClick={openCreateGroupDialog} className="px-2">
                <FolderPlus className="w-4 h-4" />
              </Button>
            </div>
          }
        />

        <MobileContent padding={false}>
          {selectionMode && (
            <div className="px-4 pt-4 pb-2 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <div className="px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 text-xs">
                已选 {selectedAccountIds.size} 个邮箱
              </div>
              <Button size="icon" variant="secondary" onClick={clearAccountSelection} className="h-8 w-8">
                <X className="w-4 h-4" />
              </Button>
            </div>
          )}

          {accounts.length > 0 ? (
            orderedGroups.length > 0 ? (
              <div className="p-4 space-y-3">
                {orderedGroups.map((group) => {
                  const accountsInGroup = accountsByGroup.get(group.id) || [];
                  const allSelected =
                    accountsInGroup.length > 0 &&
                    accountsInGroup.every((account) => selectedAccountIds.has(account.id));

                  return (
                    <EmailGroupCard
                      key={group.id}
                      group={group}
                      accountsCount={accountsInGroup.length}
                      collapsed={collapsedGroups.has(group.id)}
                      onToggleCollapse={() => toggleCollapse(group.id)}
                      showHandle={false}
                      draggable={false}
                      selectionMode={selectionMode}
                      allSelected={allSelected}
                      onToggleGroupSelection={() => handleToggleGroupSelection(group)}
                      subtitleSuffix={group.is_default ? '默认' : '显式排序'}
                      headerActions={renderGroupActions(group)}
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
                            onMoveGroup={handleOpenMoveGroupSheet}
                            active={selectedAccount?.id === account.id}
                            selectionMode={selectionMode}
                            selected={selectedAccountIds.has(account.id)}
                            onSelectToggle={() => toggleSelectAccount(account.id)}
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
                    onMoveGroup={handleOpenMoveGroupSheet}
                    active={selectedAccount?.id === account.id}
                    selectionMode={selectionMode}
                    selected={selectedAccountIds.has(account.id)}
                    onSelectToggle={() => toggleSelectAccount(account.id)}
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

        {selectionMode ? (
          <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-3 grid grid-cols-4 gap-2">
            <Button variant="outline" className="h-auto py-2 flex-col gap-1" onClick={handleBatchSync}>
              <RefreshCw className="w-4 h-4" />
              <span className="text-xs">同步</span>
            </Button>
            <Button variant="outline" className="h-auto py-2 flex-col gap-1" onClick={handleBatchMarkRead}>
              <CheckSquare className="w-4 h-4" />
              <span className="text-xs">已读</span>
            </Button>
            <Button variant="destructive" className="h-auto py-2 flex-col gap-1" onClick={handleBatchDelete}>
              <Trash2 className="w-4 h-4" />
              <span className="text-xs">删除</span>
            </Button>
            <Button
              variant="ghost"
              className="h-auto py-2 flex-col gap-1"
              onClick={() => {
                setSelectionMode(false);
                clearAccountSelection();
              }}
            >
              <X className="w-4 h-4" />
              <span className="text-xs">退出</span>
            </Button>
          </div>
        ) : accounts.length > 0 ? (
          <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-4">
            <div className="flex gap-3">
              <Button onClick={handleAddAccount} className="flex-1">
                <Plus className="w-4 h-4 mr-2" />
                添加邮箱
              </Button>
              <Button variant="outline" onClick={handleSettings} className="px-4">
                <Settings className="w-4 h-4" />
              </Button>
            </div>
          </div>
        ) : null}

        <EmailGroupDialog
          open={groupDialogOpen}
          mode={groupDialogMode}
          initialName={editingGroup?.name}
          onOpenChange={(open) => {
            setGroupDialogOpen(open);
            if (!open) {
              setEditingGroup(null);
            }
          }}
          onSubmit={handleSubmitGroupDialog}
        />

        <Sheet
          open={moveSheetOpen}
          onOpenChange={(open) => {
            setMoveSheetOpen(open);
            if (!open) {
              setMovingAccount(null);
            }
          }}
        >
          <SheetContent side="bottom" className="max-h-[80vh] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>
                {movingAccount ? `移动“${movingAccount.name}”到分组` : '移动邮箱到分组'}
              </SheetTitle>
            </SheetHeader>
            <div className="px-4 pb-6 space-y-2">
              {orderedGroups.map((group) => {
                const targetGroupId = group.is_default ? defaultGroup?.id : group.id;
                const isCurrentGroup = movingAccount?.group_id
                  ? movingAccount.group_id === targetGroupId
                  : group.is_default;

                return (
                  <button
                    key={group.id}
                    type="button"
                    onClick={() => handleMoveAccountConfirm(group)}
                    disabled={isCurrentGroup}
                    className="w-full flex items-center justify-between rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-900/40 disabled:opacity-50"
                  >
                    <div>
                      <div className="font-medium text-gray-900 dark:text-gray-100">{group.name}</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {group.is_default ? '默认分组' : `${group.account_count} 个邮箱`}
                      </div>
                    </div>
                    {isCurrentGroup && <span className="text-xs text-blue-600">当前所在</span>}
                  </button>
                );
              })}
            </div>
          </SheetContent>
        </Sheet>
      </MobilePage>
    </MobileLayout>
  );
}
