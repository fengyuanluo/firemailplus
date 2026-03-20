'use client';

import { useEffect, useCallback, useMemo, useState } from 'react';
import { SidebarHeader } from './sidebar-header';
import { AccountItem } from './account-item';
import { ContextMenu } from './context-menu';
import { AccountSettingsModal } from './account-settings-modal';
import { EmailGroupCard } from './email-group-card';
import { EmailGroupDialog } from './email-group-dialog';
import { toast } from 'sonner';
import { useContextMenuStore, useMailboxStore } from '@/lib/store';
import {
  canDeleteEmailGroup,
  canEditEmailGroup,
  canReorderEmailGroup,
  canSetDefaultEmailGroup,
  isHiddenSystemEmailGroup,
  type EmailAccount,
  type EmailGroup,
} from '@/types/email';
import { useEmailGroupActions } from '@/hooks/use-email-group-actions';
import { Button } from '@/components/ui/button';
import { Plus, RefreshCw, Trash2, X, CheckSquare } from 'lucide-react';

export function LeftSidebar() {
  const {
    accounts,
    groups,
    setGroups,
    selectionMode,
    setSelectionMode,
    selectedAccountIds,
    toggleSelectAccount,
    clearAccountSelection,
    setSelectedAccountIds,
  } = useMailboxStore();
  const { openMenu, closeMenu } = useContextMenuStore();
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
  const [settingsAccount, setSettingsAccount] = useState<EmailAccount | null>(null);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [groupModalMode, setGroupModalMode] = useState<'create' | 'edit'>('create');
  const [editingGroup, setEditingGroup] = useState<EmailGroup | null>(null);
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [draggingAccountId, setDraggingAccountId] = useState<number | null>(null);
  const [draggingGroupId, setDraggingGroupId] = useState<number | null>(null);
  const [dragOverGroupId, setDragOverGroupId] = useState<number | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<number>>(new Set());

  const loadGroupsWithState = useCallback(async () => {
    setLoadingGroups(true);
    try {
      await loadGroups();
    } catch (error) {
      console.error('Failed to load groups:', error);
    } finally {
      setLoadingGroups(false);
    }
  }, [loadGroups]);

  useEffect(() => {
    if (groups.length === 0) {
      loadGroupsWithState();
    }
  }, [groups.length, loadGroupsWithState]);

  useEffect(() => {
    if (accounts.length === 0) {
      void loadAccounts();
    }
  }, [accounts.length, loadAccounts]);

  const defaultGroup = useMemo(() => groups.find((g) => g.is_default), [groups]);
  const defaultGroupName = useMemo(() => defaultGroup?.name ?? '默认分组', [defaultGroup]);

  const orderedGroups = useMemo(() => {
    const sorted = [...groups]
      .filter((group) => !isHiddenSystemEmailGroup(group))
      .sort((a, b) => a.sort_order - b.sort_order);
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

  const displayGroups = orderedGroups.filter((g) => {
    if (isHiddenSystemEmailGroup(g)) return false;
    return true;
  });

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

  const handleCloseSettings = () => {
    setIsSettingsOpen(false);
    setSettingsAccount(null);
  };

  const handleDeleteAccount = async (targetId: number) => {
    const account = accounts.find((acc) => acc.id === targetId);
    if (!account) return;

    const confirmed = confirm(
      `确定要删除账户 "${account.email}" 吗？此操作不可撤销，将删除该账户的所有邮件数据。`
    );
    if (!confirmed) return;

    try {
      await deleteAccount(account);
    } catch (error: unknown) {
      console.error('Delete account failed:', error);
      toast.error(getErrorMessage(error, '删除账户失败'));
    }
  };

  const openGroupDialog = (mode: 'create' | 'edit', group?: EmailGroup) => {
    setGroupModalMode(mode);
    setEditingGroup(group || null);
    setGroupModalOpen(true);
  };

  const handleSubmitGroupDialog = async (name: string) => {
    try {
      if (groupModalMode === 'create') {
        await createGroup(name);
      } else if (editingGroup) {
        await renameGroup(editingGroup, name);
      }
      setEditingGroup(null);
    } catch (error: unknown) {
      console.error('Group save failed:', error);
      toast.error(getErrorMessage(error, '分组保存失败'));
    }
  };

  const handleDeleteGroup = async (group: EmailGroup) => {
    if (!canDeleteEmailGroup(group)) {
      toast.error(group.is_default ? '默认分组不可删除' : '系统分组不可删除');
      return;
    }
    const confirmed = confirm(
      `删除分组 "${group.name}" 后其中邮箱将移动到默认分组“${defaultGroupName}”，确认删除？`
    );
    if (!confirmed) return;
    try {
      await deleteGroup(group, defaultGroupName);
    } catch (error: unknown) {
      console.error('Delete group failed:', error);
      toast.error(getErrorMessage(error, '删除分组失败'));
    }
  };

  const handleMoveAccountToGroup = async (accountId: number, targetGroupId: number | null) => {
    try {
      await moveAccountToGroup(accountId, targetGroupId);
    } catch (error: unknown) {
      console.error('Move account failed:', error);
      toast.error(getErrorMessage(error, '移动邮箱到分组失败'));
    }
  };

  const reorderGroupsLocal = (sourceId: number, targetId: number): EmailGroup[] => {
    const nonDefault = orderedGroups.filter((group) => canReorderEmailGroup(group));
    const sourceIndex = nonDefault.findIndex((g) => g.id === sourceId);
    const targetIndex = nonDefault.findIndex((g) => g.id === targetId);
    if (sourceIndex < 0 || targetIndex < 0) return orderedGroups;
    const updated = [...nonDefault];
    const [moved] = updated.splice(sourceIndex, 1);
    updated.splice(targetIndex, 0, moved);
    const normalized = updated.map((g, idx) => ({ ...g, sort_order: idx + 1 }));
    return defaultGroup ? [defaultGroup, ...normalized] : normalized;
  };

  const handlePersistReorder = async (ordered: EmailGroup[]) => {
    setGroups(ordered);
    try {
      const ids = ordered.filter((group) => canReorderEmailGroup(group)).map((group) => group.id);
      await persistGroupOrder(ids);
    } catch (error: unknown) {
      console.error('Reorder groups failed:', error);
      toast.error(getErrorMessage(error, '分组排序保存失败'));
      void loadGroupsWithState();
    }
  };

  const handleGroupDrop = async (group: EmailGroup) => {
    if (draggingAccountId !== null) {
      await handleMoveAccountToGroup(draggingAccountId, group.is_default ? null : group.id);
    } else if (draggingGroupId && draggingGroupId !== group.id && canReorderEmailGroup(group)) {
      const updatedOrder = reorderGroupsLocal(draggingGroupId, group.id);
      await handlePersistReorder(updatedOrder);
    }
    setDraggingAccountId(null);
    setDraggingGroupId(null);
    setDragOverGroupId(null);
  };

  const handleBatchSync = async () => {
    const ids = Array.from(selectedAccountIds);
    if (ids.length === 0) {
      toast.error('请先选择要同步的邮箱');
      return;
    }
    try {
      await batchSyncAccounts(ids);
    } catch (error: unknown) {
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
    } catch (error: unknown) {
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
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, '批量删除失败'));
    }
  };

  const handleToggleGroupSelection = (group: EmailGroup) => {
    const accountsInGroup = accountsByGroup.get(group.id) || [];
    if (accountsInGroup.length === 0) return;
    const ids = accountsInGroup.map((a) => a.id);
    const allSelected = ids.every((id) => selectedAccountIds.has(id));
    const next = new Set(selectedAccountIds);
    if (allSelected) {
      ids.forEach((id) => next.delete(id));
    } else {
      ids.forEach((id) => next.add(id));
    }
    setSelectedAccountIds(Array.from(next));
  };

  const handleSetDefaultGroup = async (group: EmailGroup) => {
    if (!canSetDefaultEmailGroup(group)) return;
    const confirmed = confirm(`是否将分组“${group.name}”设为默认分组？`);
    if (!confirmed) return;
    try {
      await setDefaultGroup(group);
    } catch (error: unknown) {
      console.error('Set default group failed:', error);
      toast.error(getErrorMessage(error, '设置默认分组失败'));
    }
  };

  const handleBlankContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    openMenu(
      { x: e.clientX, y: e.clientY },
      {
        type: 'group-blank',
        id: 0,
      }
    );
  };

  const handleGroupContextMenu = (e: React.MouseEvent, group: EmailGroup) => {
    e.preventDefault();
    e.stopPropagation();
    if (
      !canDeleteEmailGroup(group) &&
      !canSetDefaultEmailGroup(group) &&
      !canEditEmailGroup(group)
    ) {
      closeMenu();
      return;
    }
    openMenu(
      { x: e.clientX, y: e.clientY },
      {
        type: 'group',
        id: group.id,
        data: group,
      }
    );
  };

  const handleGroupDragStart = (e: React.DragEvent, group: EmailGroup) => {
    if (!canReorderEmailGroup(group)) return;
    setDraggingGroupId(group.id);
    setDragOverGroupId(group.id);
    e.dataTransfer.setData('text/plain', String(group.id));
    e.dataTransfer.effectAllowed = 'move';
  };

  const renderGroup = (group: EmailGroup) => {
    const accountsInGroup = accountsByGroup.get(group.id) || [];
    const isDropTarget = dragOverGroupId === group.id;
    const collapsed = collapsedGroups.has(group.id);
    const allSelected =
      accountsInGroup.length > 0 && accountsInGroup.every((a) => selectedAccountIds.has(a.id));
    return (
      <EmailGroupCard
        key={group.id}
        group={group}
        accountsCount={accountsInGroup.length}
        isDropTarget={isDropTarget}
        collapsed={collapsed}
        selectionMode={selectionMode}
        allSelected={allSelected}
        onToggleCollapse={() => toggleCollapse(group.id)}
        onToggleGroupSelection={() => handleToggleGroupSelection(group)}
        onDragOver={() => setDragOverGroupId(group.id)}
        onDrop={() => handleGroupDrop(group)}
        onDragEnd={() => {
          setDraggingGroupId(null);
          setDragOverGroupId(null);
        }}
        onContextMenu={(e) => handleGroupContextMenu(e, group)}
        draggable={canReorderEmailGroup(group)}
        onDragStart={(e) => handleGroupDragStart(e, group)}
      >
        {accountsInGroup.length === 0 ? (
          <div className="text-xs text-gray-500 dark:text-gray-400 border border-dashed border-gray-300 dark:border-gray-700 rounded-md p-3">
            将邮箱拖拽到此分组
          </div>
        ) : (
          accountsInGroup.map((account) => (
            <AccountItem
              key={account.id}
              account={account}
              draggable={!selectionMode}
              selectionMode={selectionMode}
              selected={selectedAccountIds.has(account.id)}
              onSelectToggle={() => toggleSelectAccount(account.id)}
              onDragStart={(e) => {
                if (!e) return;
                e.dataTransfer.setData('text/plain', String(account.id));
                e.dataTransfer.effectAllowed = 'move';
                setDraggingAccountId(account.id);
                setDragOverGroupId(group.id);
              }}
              onDragEnd={() => {
                setDraggingAccountId(null);
                setDragOverGroupId(null);
              }}
            />
          ))
        )}
      </EmailGroupCard>
    );
  };

  return (
    <div className="h-full flex flex-col" onContextMenu={handleBlankContextMenu}>
      <SidebarHeader />

      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          邮箱分组
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant={selectionMode ? 'default' : 'ghost'}
            className="text-gray-600 dark:text-gray-300"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              if (selectionMode) {
                setSelectionMode(false);
                clearAccountSelection();
              } else {
                setSelectionMode(true);
              }
            }}
          >
            {selectionMode ? '退出选择' : '选择'}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="text-gray-600 dark:text-gray-300"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              openGroupDialog('create');
            }}
          >
            <Plus className="w-4 h-4 mr-1" />
            新建
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {selectionMode && (
          <div className="px-4 pb-2 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <div className="px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 text-xs">
              已选 {selectedAccountIds.size} 个邮箱
            </div>
            <Button
              size="icon"
              variant="secondary"
              onClick={handleBatchSync}
              disabled={selectedAccountIds.size === 0}
              title="批量同步"
            >
              <RefreshCw className="w-4 h-4" />
            </Button>
            <Button
              size="icon"
              variant="secondary"
              onClick={handleBatchMarkRead}
              disabled={selectedAccountIds.size === 0}
              title="批量已读"
            >
              <CheckSquare className="w-4 h-4" />
            </Button>
            <Button
              size="icon"
              variant="destructive"
              onClick={handleBatchDelete}
              disabled={selectedAccountIds.size === 0}
              title="批量删除"
            >
              <Trash2 className="w-4 h-4" />
            </Button>
            <Button size="icon" variant="ghost" onClick={clearAccountSelection} title="清空选择">
              <X className="w-4 h-4" />
            </Button>
          </div>
        )}
        <div className="p-4 space-y-3" onContextMenu={handleBlankContextMenu}>
          {loadingGroups && (
            <div className="text-sm text-gray-500 dark:text-gray-400">加载分组中...</div>
          )}

          {displayGroups.length > 0 ? (
            displayGroups.map((group) => renderGroup(group))
          ) : (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400 text-sm">
              右键空白区域或点击“新建”快速创建分组
            </div>
          )}
        </div>
      </div>

      <ContextMenu
        onSettings={(id) => {
          const account = accounts.find((acc) => acc.id === id);
          if (account) {
            setSettingsAccount(account);
            setIsSettingsOpen(true);
          }
        }}
        onDeleteAccount={handleDeleteAccount}
        onCreateGroup={() => openGroupDialog('create')}
        onEditGroup={(id) => {
          const target = groups.find((g) => g.id === id);
          if (target) openGroupDialog('edit', target);
        }}
        onDeleteGroup={(id) => {
          const target = groups.find((g) => g.id === id);
          if (target) handleDeleteGroup(target);
        }}
        onSetDefaultGroup={(id) => {
          const target = groups.find((g) => g.id === id);
          if (target) handleSetDefaultGroup(target);
        }}
      />

      <AccountSettingsModal
        isOpen={isSettingsOpen}
        onClose={handleCloseSettings}
        account={settingsAccount}
      />

      <EmailGroupDialog
        open={groupModalOpen}
        mode={groupModalMode}
        initialName={editingGroup?.name}
        onOpenChange={(open) => {
          setGroupModalOpen(open);
          if (!open) {
            setEditingGroup(null);
          }
        }}
        onSubmit={handleSubmitGroupDialog}
      />
    </div>
  );
}
