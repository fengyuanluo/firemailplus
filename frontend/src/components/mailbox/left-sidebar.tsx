'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import type { DragEvent as ReactDragEvent, MouseEvent as ReactMouseEvent } from 'react';
import { SidebarHeader } from './sidebar-header';
import { AccountItem } from './account-item';
import { ContextMenu } from './context-menu';
import { AccountSettingsModal } from './account-settings-modal';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { useMailboxStore, useContextMenuStore } from '@/lib/store';
import type { EmailAccount, EmailAccountGroup } from '@/types/email';
import { ChevronDown, ChevronRight, ChevronUp } from 'lucide-react';

type GroupKey = number | null;

const ACCOUNT_DRAG_MIME = 'application/x-account-ids';
const GROUP_DRAG_MIME = 'application/x-group-id';
const COLLAPSED_GROUPS_STORAGE_KEY = 'mailboxCollapsedGroups';

export function LeftSidebar() {
  const { accounts, accountGroups, setAccounts, setAccountGroups, removeAccount } =
    useMailboxStore();
  const { openMenu } = useContextMenuStore();

  const [settingsAccount, setSettingsAccount] = useState<EmailAccount | null>(null);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [selectedAccountIds, setSelectedAccountIds] = useState<Set<number>>(new Set());
  const [draggingAccountIds, setDraggingAccountIds] = useState<number[]>([]);
  const [dragOverAccountId, setDragOverAccountId] = useState<number | null>(null);
  const [dragOverGroupId, setDragOverGroupId] = useState<GroupKey>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [pendingMoveGroup, setPendingMoveGroup] = useState<string>('');
  const [collapsedGroups, setCollapsedGroups] = useState<Set<GroupKey>>(new Set());
  const [draggingGroupForSortId, setDraggingGroupForSortId] = useState<number | null>(null);
  const [groupSortHoverId, setGroupSortHoverId] = useState<number | 'end' | null>(null);

  const isAccountDragEvent = useCallback(
    (event: ReactDragEvent<HTMLElement>) =>
      event.dataTransfer.types.includes(ACCOUNT_DRAG_MIME) ||
      event.dataTransfer.types.includes('text/plain'),
    []
  );

  const loadAccountGroups = useCallback(
    async (force = false) => {
      if (!force && accountGroups.length > 0) return;
      try {
        const response = await apiClient.getAccountGroups();
        if (response.success && response.data) {
          setAccountGroups(response.data);
        }
      } catch (error) {
        console.error('Failed to load account groups:', error);
      }
    },
    [accountGroups.length, setAccountGroups]
  );

  const loadAccounts = useCallback(
    async (force = false) => {
      if (!force && accounts.length > 0) return;
      try {
        const response = await apiClient.getEmailAccounts();
        if (response.success && response.data) {
          setAccounts(response.data);
        }
      } catch (error) {
        console.error('Failed to load accounts:', error);
      }
    },
    [accounts.length, setAccounts]
  );

  useEffect(() => {
    loadAccountGroups();
  }, [loadAccountGroups]);

  useEffect(() => {
    loadAccounts();
  }, [loadAccounts]);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      const stored = window.localStorage.getItem(COLLAPSED_GROUPS_STORAGE_KEY);
      if (!stored) return;
      const parsed: Array<number | 'null'> = JSON.parse(stored);
      const restored = new Set<GroupKey>();
      parsed.forEach((item) => {
        if (item === 'null') {
          restored.add(null);
        } else {
          restored.add(Number(item));
        }
      });
      setCollapsedGroups(restored);
    } catch (error) {
      console.error('Failed to restore collapsed groups:', error);
    }
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      const serialized = JSON.stringify(
        Array.from(collapsedGroups).map((item) => (item === null ? 'null' : item))
      );
      window.localStorage.setItem(COLLAPSED_GROUPS_STORAGE_KEY, serialized);
    } catch (error) {
      console.error('Failed to persist collapsed groups:', error);
    }
  }, [collapsedGroups]);

  useEffect(() => {
    if (selectedAccountIds.size === 0) {
      return;
    }
    const existingIds = new Set(accounts.map((account) => account.id));
    const filtered = new Set<number>();
    selectedAccountIds.forEach((id) => {
      if (existingIds.has(id)) {
        filtered.add(id);
      }
    });
    if (filtered.size !== selectedAccountIds.size) {
      setSelectedAccountIds(filtered);
    }
  }, [accounts, selectedAccountIds]);

  const groupedAccounts = useMemo(() => {
    const map = new Map<GroupKey, EmailAccount[]>();
    map.set(null, []);
    accountGroups.forEach((group) => {
      map.set(group.id, []);
    });

    accounts.forEach((account) => {
      const key = account.group_id ?? null;
      if (!map.has(key)) {
        map.set(key, []);
      }
      map.get(key)!.push(account);
    });

    return map;
  }, [accounts, accountGroups]);

  const handleReloadData = useCallback(async () => {
    setIsRefreshing(true);
    try {
      await Promise.all([loadAccounts(true), loadAccountGroups(true)]);
    } finally {
      setIsRefreshing(false);
    }
  }, [loadAccounts, loadAccountGroups]);

  const handleAccountSelection = useCallback(
    (account: EmailAccount, event: ReactMouseEvent<HTMLDivElement>) => {
      setSelectedAccountIds((prev) => {
        const next = new Set(prev);
        const shouldToggle = event.metaKey || event.ctrlKey || prev.size > 0;
        if (shouldToggle) {
          if (next.has(account.id)) {
            next.delete(account.id);
          } else {
            next.add(account.id);
          }
          return next.size > 0 ? next : new Set([account.id]);
        }
        return new Set([account.id]);
      });
    },
    []
  );

  const handleCreateGroup = useCallback(async () => {
    const name = prompt('请输入分组名称');
    if (!name) return;
    try {
      const response = await apiClient.createAccountGroup({ name });
      if (response.success && response.data) {
        toast.success('分组已创建');
        await loadAccountGroups(true);
      } else {
        throw new Error(response.message || '创建失败');
      }
    } catch (error: any) {
      console.error('Create group failed:', error);
      toast.error(error.message || '创建分组失败');
    }
  }, [loadAccountGroups]);

  const handleRenameGroup = useCallback(
    async (groupId: number) => {
      const group = accountGroups.find((item) => item.id === groupId);
      if (!group) return;
      const name = prompt('修改分组名称', group.name);
      if (!name || name === group.name) return;
      try {
        const response = await apiClient.updateAccountGroup(groupId, { name });
        if (response.success && response.data) {
          toast.success('分组已更新');
          await loadAccountGroups(true);
        } else {
          throw new Error(response.message || '更新失败');
        }
      } catch (error: any) {
        console.error('Rename group failed:', error);
        toast.error(error.message || '更新分组失败');
      }
    },
    [accountGroups, loadAccountGroups]
  );

  const handleDeleteGroup = useCallback(
    async (groupId: number) => {
      const group = accountGroups.find((item) => item.id === groupId);
      if (!group) return;
      const confirmed = confirm(`确定删除分组“${group.name}”吗？该分组内的邮箱将移到未分组。`);
      if (!confirmed) return;
      try {
        const response = await apiClient.deleteAccountGroup(groupId);
        if (response.success) {
          toast.success('分组已删除');
          await handleReloadData();
        } else {
          throw new Error(response.message || '删除失败');
        }
      } catch (error: any) {
        console.error('Delete group failed:', error);
        toast.error(error.message || '删除分组失败');
      }
    },
    [accountGroups, handleReloadData]
  );

  const handleGroupMove = useCallback(
    async (group: EmailAccountGroup, direction: -1 | 1) => {
      const currentIndex = accountGroups.findIndex((item) => item.id === group.id);
      const targetIndex = currentIndex + direction;
      if (targetIndex < 0 || targetIndex >= accountGroups.length) return;

      const reordered = [...accountGroups];
      const [removed] = reordered.splice(currentIndex, 1);
      reordered.splice(targetIndex, 0, removed);

      const orders = reordered.map((item, index) => ({ id: item.id, sort_order: index }));
      try {
        const response = await apiClient.reorderAccountGroups(orders);
        if (response.success) {
          await loadAccountGroups(true);
        } else {
          throw new Error(response.message || '排序失败');
        }
      } catch (error: any) {
        console.error('Reorder groups failed:', error);
        toast.error(error.message || '调整分组排序失败');
      }
    },
    [accountGroups, loadAccountGroups]
  );

  const toggleGroupCollapse = useCallback((groupId: GroupKey) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
      }
      return next;
    });
  }, []);

  const resetGroupSortDragState = useCallback(() => {
    setDraggingGroupForSortId(null);
    setGroupSortHoverId(null);
  }, []);

  const performGroupReorder = useCallback(
    async (sourceId: number, target: number | 'end') => {
      if (sourceId === target) {
        resetGroupSortDragState();
        return;
      }

      const sourceIndex = accountGroups.findIndex((item) => item.id === sourceId);
      if (sourceIndex === -1) {
        resetGroupSortDragState();
        return;
      }

      const ordered = [...accountGroups];
      const [removed] = ordered.splice(sourceIndex, 1);

      let insertIndex: number;

      if (target === 'end') {
        insertIndex = ordered.length;
      } else {
        insertIndex = ordered.findIndex((item) => item.id === target);
        if (insertIndex === -1) {
          resetGroupSortDragState();
          return;
        }
        if (sourceIndex < insertIndex) {
          insertIndex -= 1;
        }
      }

      ordered.splice(insertIndex, 0, removed);

      const orders = ordered.map((item, index) => ({ id: item.id, sort_order: index }));

      try {
        const response = await apiClient.reorderAccountGroups(orders);
        if (!response.success) {
          throw new Error(response.message || '排序失败');
        }
        await loadAccountGroups(true);
      } catch (error: any) {
        console.error('Reorder groups failed:', error);
        toast.error(error.message || '调整分组排序失败');
      } finally {
        resetGroupSortDragState();
      }
    },
    [accountGroups, loadAccountGroups, resetGroupSortDragState]
  );

  const handleGroupSortDragStart = useCallback(
    (groupId: number, event: ReactDragEvent<HTMLDivElement>) => {
      event.stopPropagation();
      setDraggingGroupForSortId(groupId);
      setGroupSortHoverId(groupId);
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData(GROUP_DRAG_MIME, groupId.toString());
    },
    []
  );

  const handleGroupSortDragEnter = useCallback(
    (groupId: number, event: ReactDragEvent<HTMLDivElement>) => {
      if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
      if (draggingGroupForSortId === groupId) return;
      event.preventDefault();
      setGroupSortHoverId(groupId);
    },
    [draggingGroupForSortId]
  );

  const handleGroupSortDragOver = useCallback((event: ReactDragEvent<HTMLDivElement>) => {
    if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const handleGroupSortDragEnd = useCallback(() => {
    resetGroupSortDragState();
  }, [resetGroupSortDragState]);

  const handleGroupSortDrop = useCallback(
    (groupId: number, event: ReactDragEvent<HTMLDivElement>) => {
      if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
      event.preventDefault();
      event.stopPropagation();
      const source = Number(event.dataTransfer.getData(GROUP_DRAG_MIME));
      if (!source || source === groupId) {
        resetGroupSortDragState();
        return;
      }
      performGroupReorder(source, groupId);
    },
    [performGroupReorder, resetGroupSortDragState]
  );

  const handleGroupSortDropToEnd = useCallback(
    (event: ReactDragEvent<HTMLDivElement>) => {
      if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
      event.preventDefault();
      event.stopPropagation();
      const source = Number(event.dataTransfer.getData(GROUP_DRAG_MIME));
      if (!source) {
        resetGroupSortDragState();
        return;
      }
      performGroupReorder(source, 'end');
    },
    [performGroupReorder, resetGroupSortDragState]
  );

  const handleGroupSortDragLeave = useCallback(
    (groupId: number, event: ReactDragEvent<HTMLDivElement>) => {
      if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
      if (groupSortHoverId === groupId) {
        setGroupSortHoverId(null);
      }
    },
    [groupSortHoverId]
  );

  const handleAccountDragStart = useCallback(
    (account: EmailAccount, event: ReactDragEvent<HTMLDivElement>) => {
      const currentSelection = selectedAccountIds.has(account.id)
        ? new Set(selectedAccountIds)
        : new Set<number>([account.id]);
      setSelectedAccountIds(currentSelection);
      const movingIds = Array.from(currentSelection);
      setDraggingAccountIds(movingIds);
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData(ACCOUNT_DRAG_MIME, movingIds.join(','));
      event.dataTransfer.setData('text/plain', movingIds.join(','));
    },
    [selectedAccountIds]
  );

  const handleAccountDragEnter = useCallback(
    (account: EmailAccount, event: ReactDragEvent<HTMLDivElement>) => {
      if (!isAccountDragEvent(event)) return;
      if (draggingAccountIds.includes(account.id)) return;
      event.preventDefault();
      setDragOverAccountId(account.id);
      setDragOverGroupId(account.group_id ?? null);
    },
    [draggingAccountIds, isAccountDragEvent]
  );

  const handleAccountDragOver = useCallback(
    (event: ReactDragEvent<HTMLDivElement>) => {
      if (!isAccountDragEvent(event)) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = 'move';
    },
    [isAccountDragEvent]
  );

  const resetDragState = useCallback(() => {
    setDraggingAccountIds([]);
    setDragOverAccountId(null);
    setDragOverGroupId(null);
  }, []);

  const performDrop = useCallback(
    async (targetGroupId: GroupKey, beforeAccountId: number | null) => {
      const movingIds =
        draggingAccountIds.length > 0 ? draggingAccountIds : Array.from(selectedAccountIds);
      if (movingIds.length === 0) {
        resetDragState();
        return;
      }

      const accountMap = new Map(accounts.map((account) => [account.id, account]));
      const firstAccount = accountMap.get(movingIds[0]);
      if (!firstAccount) {
        resetDragState();
        return;
      }

      const sourceGroupId = firstAccount.group_id ?? null;
      if (beforeAccountId && movingIds.includes(beforeAccountId)) {
        resetDragState();
        return;
      }

      const localGrouped = new Map<GroupKey, EmailAccount[]>();
      groupedAccounts.forEach((value, key) => {
        localGrouped.set(key, [...value]);
      });

      localGrouped.forEach((value, key) => {
        localGrouped.set(
          key,
          value.filter((account) => !movingIds.includes(account.id))
        );
      });

      const movingAccounts = movingIds
        .map((id) => accountMap.get(id))
        .filter((account): account is EmailAccount => Boolean(account));

      const targetArray = localGrouped.get(targetGroupId) ?? [];
      const targetIndex = beforeAccountId
        ? targetArray.findIndex((account) => account.id === beforeAccountId)
        : -1;
      const insertIndex = targetIndex >= 0 ? targetIndex : targetArray.length;

      targetArray.splice(insertIndex, 0, ...movingAccounts);
      localGrouped.set(targetGroupId, targetArray);

      const affectedGroups = new Set<GroupKey>();
      affectedGroups.add(targetGroupId);
      if (sourceGroupId !== targetGroupId) {
        affectedGroups.add(sourceGroupId);
      }

      const orders: { account_id: number; sort_order: number }[] = [];
      affectedGroups.forEach((groupId) => {
        const list = localGrouped.get(groupId) ?? [];
        list.forEach((account, index) => {
          orders.push({ account_id: account.id, sort_order: index });
        });
      });

      try {
        if (sourceGroupId !== targetGroupId) {
          await apiClient.moveAccountsToGroup({
            account_ids: movingIds,
            group_id: targetGroupId,
          });
        }

        if (orders.length > 0) {
          await apiClient.reorderAccounts(orders);
        }

        await handleReloadData();
        setSelectedAccountIds(new Set(movingIds));
      } catch (error: any) {
        console.error('Move account failed:', error);
        toast.error(error.message || '移动邮箱失败');
      } finally {
        resetDragState();
      }
    },
    [
      accounts,
      draggingAccountIds,
      groupedAccounts,
      handleReloadData,
      resetDragState,
      selectedAccountIds,
    ]
  );

  const handleAccountDrop = useCallback(
    (account: EmailAccount, event: ReactDragEvent<HTMLDivElement>) => {
      event.preventDefault();
      event.stopPropagation();
      performDrop(account.group_id ?? null, account.id);
    },
    [performDrop]
  );

  const handleGroupDragEnter = useCallback(
    (groupId: GroupKey, event: ReactDragEvent<HTMLDivElement>) => {
      if (!isAccountDragEvent(event)) return;
      event.preventDefault();
      setDragOverGroupId(groupId);
      setDragOverAccountId(null);
    },
    [isAccountDragEvent]
  );

  const handleGroupDragOver = useCallback(
    (event: ReactDragEvent<HTMLDivElement>, groupId: GroupKey) => {
      if (!isAccountDragEvent(event)) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = 'move';
      setDragOverGroupId(groupId);
      setDragOverAccountId(null);
    },
    [isAccountDragEvent]
  );

  const handleGroupDrop = useCallback(
    (groupId: GroupKey, event: ReactDragEvent<HTMLDivElement>) => {
      if (!isAccountDragEvent(event)) return;
      event.preventDefault();
      event.stopPropagation();
      performDrop(groupId, null);
    },
    [isAccountDragEvent, performDrop]
  );

  const handleMoveSelectedAccounts = useCallback(async () => {
    const accountIds = Array.from(selectedAccountIds);
    if (accountIds.length === 0) return;

    const targetGroupId = pendingMoveGroup === '' ? null : Number(pendingMoveGroup);

    try {
      const response = await apiClient.moveAccountsToGroup({
        account_ids: accountIds,
        group_id: targetGroupId,
      });
      if (!response.success) {
        throw new Error(response.message || '移动失败');
      }
      toast.success('邮箱已移动');
      await handleReloadData();
      setSelectedAccountIds(new Set(accountIds));
      setPendingMoveGroup('');
    } catch (error: any) {
      console.error('Move accounts failed:', error);
      toast.error(error.message || '移动邮箱失败');
    }
  }, [handleReloadData, pendingMoveGroup, selectedAccountIds]);

  const handleSettings = useCallback(
    (targetId: number) => {
      const account = accounts.find((item) => item.id === targetId);
      if (account) {
        setSettingsAccount(account);
        setIsSettingsOpen(true);
      }
    },
    [accounts]
  );

  const handleDeleteAccount = useCallback(
    async (targetId: number) => {
      const account = accounts.find((item) => item.id === targetId);
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
    },
    [accounts, removeAccount]
  );

  const renderGroupHeader = useCallback(
    (
      group: EmailAccountGroup | null,
      accountsInGroup: EmailAccount[],
      isActiveDrop: boolean,
      groupId: GroupKey
    ) => {
      const isDefault = group === null;
      const label = isDefault ? '未分组' : group!.name;
      const isCollapsed = collapsedGroups.has(groupId);
      const canSortGroup = !isDefault && accountGroups.length > 1;
      const isSortHover = !isDefault && groupSortHoverId === group?.id;
      const isDraggingGroup = !isDefault && draggingGroupForSortId === group?.id;

      const handleContextMenu = (event: ReactMouseEvent<HTMLDivElement>) => {
        event.preventDefault();
        event.stopPropagation();
        const position = { x: event.clientX, y: event.clientY };
        if (isDefault) {
          openMenu(position, { type: 'sidebar', data: null });
        } else {
          openMenu(position, { type: 'group', id: group!.id, data: group });
        }
      };

      const highlightClasses = isActiveDrop
        ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-200 shadow-inner'
        : '';
      const sortHighlightClasses =
        isSortHover || isDraggingGroup
          ? 'ring-1 ring-blue-400 dark:ring-blue-500 bg-blue-50/60 dark:bg-blue-900/20 text-blue-700 dark:text-blue-200'
          : '';

      const combinedClasses = [
        'flex items-center justify-between px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider transition-colors text-gray-500 dark:text-gray-400',
        highlightClasses,
        sortHighlightClasses,
      ]
        .filter(Boolean)
        .join(' ');

      return (
        <div
          className={combinedClasses}
          onContextMenu={handleContextMenu}
          draggable={canSortGroup}
          onDragStart={
            canSortGroup ? (event) => handleGroupSortDragStart(group!.id, event) : undefined
          }
          onDragEnd={canSortGroup ? handleGroupSortDragEnd : undefined}
          onDragEnter={(event) => {
            if (event.dataTransfer.types.includes(GROUP_DRAG_MIME)) {
              if (canSortGroup) {
                handleGroupSortDragEnter(group!.id, event);
              }
              return;
            }
            handleGroupDragEnter(groupId, event);
          }}
          onDragOver={(event) => {
            if (event.dataTransfer.types.includes(GROUP_DRAG_MIME)) {
              if (canSortGroup) {
                handleGroupSortDragOver(event);
              }
              return;
            }
            handleGroupDragOver(event, groupId);
          }}
          onDragLeave={(event) => {
            if (event.dataTransfer.types.includes(GROUP_DRAG_MIME)) {
              if (canSortGroup) {
                handleGroupSortDragLeave(group!.id, event);
              }
            }
          }}
          onDrop={(event) => {
            if (event.dataTransfer.types.includes(GROUP_DRAG_MIME)) {
              if (canSortGroup) {
                handleGroupSortDrop(group!.id, event);
              }
              return;
            }
            handleGroupDrop(groupId, event);
          }}
        >
          <div className="flex items-center gap-2 overflow-hidden">
            <button
              type="button"
              className="flex-shrink-0 p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
              onClick={(event) => {
                event.stopPropagation();
                toggleGroupCollapse(groupId);
              }}
            >
              {isCollapsed ? <ChevronRight className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
            </button>
            <span className="truncate">{label}</span>
          </div>
          <div className="flex items-center gap-1 text-gray-400 dark:text-gray-500">
            <span>{accountsInGroup.length}</span>
            {!isDefault && (
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  className="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
                  onClick={() => handleGroupMove(group!, -1)}
                >
                  <ChevronUp className="w-3 h-3" />
                </button>
                <button
                  type="button"
                  className="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
                  onClick={() => handleGroupMove(group!, 1)}
                >
                  <ChevronDown className="w-3 h-3" />
                </button>
              </div>
            )}
          </div>
        </div>
      );
    },
    [
      accountGroups,
      collapsedGroups,
      draggingGroupForSortId,
      groupSortHoverId,
      handleGroupDragEnter,
      handleGroupDragOver,
      handleGroupDrop,
      handleGroupMove,
      handleGroupSortDragEnter,
      handleGroupSortDragLeave,
      handleGroupSortDragOver,
      handleGroupSortDragStart,
      handleGroupSortDragEnd,
      handleGroupSortDrop,
      openMenu,
      toggleGroupCollapse,
    ]
  );

  const renderGroupSection = (group: EmailAccountGroup | null) => {
    const groupId = group?.id ?? null;
    const accountsInGroup = groupedAccounts.get(groupId) ?? [];
    const isDragTarget = dragOverGroupId === groupId && dragOverAccountId === null;
    const isCollapsed = collapsedGroups.has(groupId);

    return (
      <div
        key={groupId ?? 'ungrouped'}
        className="space-y-2 rounded-lg"
        onDragEnter={(event) => handleGroupDragEnter(groupId, event)}
        onDragOver={(event) => handleGroupDragOver(event, groupId)}
        onDrop={(event) => handleGroupDrop(groupId, event)}
      >
        {renderGroupHeader(group, accountsInGroup, isDragTarget, groupId)}
        {!isCollapsed && accountsInGroup.length > 0 && (
          <div className="space-y-2">
            {accountsInGroup.map((account) => (
              <AccountItem
                key={account.id}
                account={account}
                isSelected={selectedAccountIds.has(account.id)}
                isDragOver={dragOverAccountId === account.id}
                onAccountClick={(event) => handleAccountSelection(account, event)}
                onAccountDragStart={(event) => handleAccountDragStart(account, event)}
                onAccountDragEnter={(event) => handleAccountDragEnter(account, event)}
                onAccountDragOver={handleAccountDragOver}
                onAccountDrop={(event) => handleAccountDrop(account, event)}
                onAccountDragEnd={resetDragState}
              />
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div
      className="h-full flex flex-col"
      onContextMenu={(event) => {
        event.preventDefault();
        event.stopPropagation();
        openMenu({ x: event.clientX, y: event.clientY }, { type: 'sidebar', data: null });
      }}
    >
      <SidebarHeader />

      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-6">
          <div className="flex items-center justify-between text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
            <span>邮箱分组</span>
            <button
              type="button"
              className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
              onClick={handleCreateGroup}
            >
              新建分组
            </button>
          </div>
          {accounts.length > 0 ? (
            <>
              {renderGroupSection(null)}
              {accountGroups.map((group) => renderGroupSection(group))}
              {draggingGroupForSortId !== null && accountGroups.length > 0 && (
                <div
                  className={`h-8 border border-dashed rounded-md flex items-center justify-center text-xs transition-colors ${
                    groupSortHoverId === 'end'
                      ? 'border-blue-400 dark:border-blue-500 text-blue-600 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-transparent text-gray-400 dark:text-gray-500'
                  }`}
                  onDragOver={(event) => handleGroupSortDragOver(event)}
                  onDragEnter={(event) => {
                    if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
                    event.preventDefault();
                    setGroupSortHoverId('end');
                  }}
                  onDragLeave={(event) => {
                    if (!event.dataTransfer.types.includes(GROUP_DRAG_MIME)) return;
                    if (groupSortHoverId === 'end') {
                      setGroupSortHoverId(null);
                    }
                  }}
                  onDrop={(event) => handleGroupSortDropToEnd(event)}
                >
                  拖动到此放到末尾
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-8">
              <div className="text-gray-500 dark:text-gray-400 text-sm">暂无邮箱账户</div>
              <div className="text-xs text-gray-400 dark:text-gray-500 mt-1">请先添加邮箱账户</div>
            </div>
          )}
          {isRefreshing && <div className="text-center text-xs text-gray-400">正在刷新...</div>}
        </div>
      </div>

      {selectedAccountIds.size > 0 && (
        <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-4 py-3 space-y-2">
          <div className="text-xs text-gray-500 dark:text-gray-400">
            已选择 {selectedAccountIds.size} 个邮箱
          </div>
          <div className="flex flex-col sm:flex-row gap-2">
            <select
              className="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm"
              value={pendingMoveGroup}
              onChange={(event) => setPendingMoveGroup(event.target.value)}
            >
              <option value="">未分组</option>
              {accountGroups.map((group) => (
                <option key={group.id} value={group.id.toString()}>
                  {group.name}
                </option>
              ))}
            </select>
            <div className="flex gap-2">
              <button
                type="button"
                className="flex-1 sm:flex-initial px-3 py-2 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-700"
                onClick={handleMoveSelectedAccounts}
              >
                移动
              </button>
              <button
                type="button"
                className="flex-1 sm:flex-initial px-3 py-2 text-sm rounded-md border border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300"
                onClick={() => {
                  setSelectedAccountIds(new Set());
                  setPendingMoveGroup('');
                }}
              >
                取消选择
              </button>
            </div>
          </div>
        </div>
      )}

      <ContextMenu
        onSettings={handleSettings}
        onDeleteAccount={handleDeleteAccount}
        onCreateGroup={handleCreateGroup}
        onRenameGroup={handleRenameGroup}
        onDeleteGroup={handleDeleteGroup}
      />

      <AccountSettingsModal
        isOpen={isSettingsOpen}
        onClose={() => {
          setIsSettingsOpen(false);
          setSettingsAccount(null);
        }}
        account={settingsAccount}
      />
    </div>
  );
}
