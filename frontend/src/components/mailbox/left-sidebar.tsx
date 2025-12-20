'use client';

import { useEffect, useCallback, useMemo, useState } from 'react';
import { SidebarHeader } from './sidebar-header';
import { AccountItem } from './account-item';
import { ContextMenu } from './context-menu';
import { AccountSettingsModal } from './account-settings-modal';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { useContextMenuStore, useMailboxStore } from '@/lib/store';
import type { EmailAccount, EmailGroup } from '@/types/email';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Plus, FolderPlus, Hash } from 'lucide-react';

export function LeftSidebar() {
  const {
    accounts,
    setAccounts,
    removeAccount,
    updateAccount,
    groups,
    setGroups,
  } = useMailboxStore();
  const { openMenu } = useContextMenuStore();
  const [settingsAccount, setSettingsAccount] = useState<EmailAccount | null>(null);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [groupModalMode, setGroupModalMode] = useState<'create' | 'edit'>('create');
  const [editingGroup, setEditingGroup] = useState<EmailGroup | null>(null);
  const [groupName, setGroupName] = useState('');
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [draggingAccountId, setDraggingAccountId] = useState<number | null>(null);
  const [draggingGroupId, setDraggingGroupId] = useState<number | null>(null);
  const [dragOverGroupId, setDragOverGroupId] = useState<number | null>(null);

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
    if (groups.length === 0) {
      loadGroups();
    }
  }, [groups.length, loadGroups]);

  useEffect(() => {
    if (accounts.length === 0) {
      loadAccounts();
    }
  }, [accounts.length, loadAccounts]);

  const defaultGroup = useMemo(() => groups.find((g) => g.is_default), [groups]);
  const defaultGroupName = useMemo(() => defaultGroup?.name ?? '默认分组', [defaultGroup]);

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

  const defaultGroupAccounts = defaultGroup ? accountsByGroup.get(defaultGroup.id) || [] : [];
  const shouldShowDefault = Boolean(
    defaultGroup && (defaultGroupAccounts.length > 0 || draggingAccountId !== null)
  );

  const displayGroups = orderedGroups.filter((g) => !g.is_default || shouldShowDefault);

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
        loadGroups();
      } else {
        throw new Error(response.message || '删除失败');
      }
    } catch (error: any) {
      console.error('Delete account failed:', error);
      toast.error(error.message || '删除账户失败');
    }
  };

  const openGroupDialog = (mode: 'create' | 'edit', group?: EmailGroup) => {
    setGroupModalMode(mode);
    setEditingGroup(group || null);
    setGroupName(group?.name ?? '');
    setGroupModalOpen(true);
  };

  const handleSubmitGroupDialog = async () => {
    const name = groupName.trim();
    if (!name) {
      toast.error('请填写分组名称');
      return;
    }

    try {
      if (groupModalMode === 'create') {
        const response = await apiClient.createEmailGroup({ name });
        if (response.success) {
          toast.success('分组已创建');
          await Promise.all([loadGroups(), loadAccounts()]);
        }
      } else if (editingGroup) {
        const response = await apiClient.updateEmailGroup(editingGroup.id, { name });
        if (response.success) {
          toast.success('分组已更新');
          await loadGroups();
        }
      }
      setGroupModalOpen(false);
      setGroupName('');
      setEditingGroup(null);
    } catch (error: any) {
      console.error('Group save failed:', error);
      toast.error(error.message || '分组保存失败');
    }
  };

  const handleDeleteGroup = async (group: EmailGroup) => {
    if (group.is_default) {
      toast.error('默认分组不可删除');
      return;
    }
    const confirmed = confirm(
      `删除分组 "${group.name}" 后其中邮箱将移动到默认分组“${defaultGroupName}”，确认删除？`
    );
    if (!confirmed) return;
    try {
      const resp = await apiClient.deleteEmailGroup(group.id);
      if (resp.success) {
        toast.success(`分组已删除，相关邮箱已移动到默认分组“${defaultGroupName}”`);
        await Promise.all([loadGroups(), loadAccounts()]);
      } else {
        throw new Error(resp.message || '删除失败');
      }
    } catch (error: any) {
      console.error('Delete group failed:', error);
      toast.error(error.message || '删除分组失败');
    }
  };

  const handleMoveAccountToGroup = async (accountId: number, targetGroupId?: number) => {
    try {
      const response = await apiClient.updateEmailAccount(accountId, {
        group_id: targetGroupId,
      });
      if (response.success && response.data) {
        updateAccount(response.data);
        await loadGroups();
      }
    } catch (error: any) {
      console.error('Move account failed:', error);
      toast.error(error.message || '移动邮箱到分组失败');
    }
  };

  const reorderGroupsLocal = (sourceId: number, targetId: number): EmailGroup[] => {
    const nonDefault = orderedGroups.filter((g) => !g.is_default);
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
    const ids = ordered.filter((g) => !g.is_default).map((g) => g.id);
    try {
      await apiClient.reorderEmailGroups(ids);
    } catch (error: any) {
      console.error('Reorder groups failed:', error);
      toast.error(error.message || '分组排序保存失败');
      loadGroups();
    }
  };

  const handleGroupDrop = async (group: EmailGroup) => {
    if (draggingAccountId !== null) {
      await handleMoveAccountToGroup(draggingAccountId, group.id);
    } else if (draggingGroupId && draggingGroupId !== group.id) {
      const updatedOrder = reorderGroupsLocal(draggingGroupId, group.id);
      await handlePersistReorder(updatedOrder);
    }
    setDraggingAccountId(null);
    setDraggingGroupId(null);
    setDragOverGroupId(null);
  };

  const handleSetDefaultGroup = async (group: EmailGroup) => {
    if (group.is_default) return;
    const confirmed = confirm(`是否将分组“${group.name}”设为默认分组？`);
    if (!confirmed) return;
    try {
      const resp = await apiClient.setDefaultEmailGroup(group.id);
      if (resp.success) {
        toast.success(`已将“${group.name}”设为默认分组`);
        await Promise.all([loadGroups(), loadAccounts()]);
      } else {
        throw new Error(resp.message || '设置默认分组失败');
      }
    } catch (error: any) {
      console.error('Set default group failed:', error);
      toast.error(error.message || '设置默认分组失败');
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
    if (group.is_default) return;
    setDraggingGroupId(group.id);
    setDragOverGroupId(group.id);
    e.dataTransfer.setData('text/plain', String(group.id));
    e.dataTransfer.effectAllowed = 'move';
  };

  const renderGroup = (group: EmailGroup) => {
    const accountsInGroup = accountsByGroup.get(group.id) || [];
    const isDropTarget = dragOverGroupId === group.id;
    return (
      <div
        key={group.id}
        className={`border rounded-lg p-3 transition-colors ${
          isDropTarget
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-200 dark:border-gray-700'
        }`}
        onDragOver={(e) => {
          e.preventDefault();
          setDragOverGroupId(group.id);
        }}
        onDrop={(e) => {
          e.preventDefault();
          handleGroupDrop(group);
        }}
        onDragEnd={() => {
          setDraggingGroupId(null);
          setDragOverGroupId(null);
        }}
        onContextMenu={(e) => handleGroupContextMenu(e, group)}
      >
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <div className="flex items-center justify-center w-8 h-8 rounded-full bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300">
                {group.is_default ? <FolderPlus className="w-4 h-4" /> : <Hash className="w-4 h-4" />}
              </div>
            <div>
              <div className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {group.name}
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">
                {accountsInGroup.length} 个邮箱
                {group.is_default ? ' · 默认' : ' · 可拖动排序'}
              </div>
            </div>
            </div>
            <div
              className={`text-xs text-gray-400 ${group.is_default ? '' : 'cursor-grab'}`}
              draggable={!group.is_default}
              onDragStart={(e) => handleGroupDragStart(e, group)}
              onDragEnd={() => {
                setDraggingGroupId(null);
                setDragOverGroupId(null);
              }}
            >
              {group.is_default ? '默认分组' : '拖动排序'}
            </div>
          </div>

        <div className="mt-2 space-y-2">
          {accountsInGroup.length === 0 ? (
            <div className="text-xs text-gray-500 dark:text-gray-400 border border-dashed border-gray-300 dark:border-gray-700 rounded-md p-3">
              将邮箱拖拽到此分组
            </div>
          ) : (
            accountsInGroup.map((account) => (
              <AccountItem
                key={account.id}
                account={account}
                draggable
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
        </div>
      </div>
    );
  };

  return (
    <div className="h-full flex flex-col" onContextMenu={handleBlankContextMenu}>
      <SidebarHeader />

      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          邮箱分组
        </div>
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

      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-3" onContextMenu={handleBlankContextMenu}>
          {loadingGroups && (
            <div className="text-sm text-gray-500 dark:text-gray-400">加载分组中...</div>
          )}

          {defaultGroup && defaultGroupAccounts.length === 0 && draggingAccountId !== null && (
            <div
              className={`border-2 border-dashed rounded-lg p-3 text-sm cursor-pointer ${
                dragOverGroupId === defaultGroup.id
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-200'
                  : 'border-gray-300 dark:border-gray-700 text-gray-600 dark:text-gray-300'
              }`}
              onDragOver={(e) => {
                e.preventDefault();
                setDragOverGroupId(defaultGroup.id);
              }}
              onDrop={(e) => {
                e.preventDefault();
                if (draggingAccountId !== null) {
                  handleMoveAccountToGroup(draggingAccountId, defaultGroup.id);
                }
                setDragOverGroupId(null);
                setDraggingAccountId(null);
              }}
              onDragLeave={() => setDragOverGroupId(null)}
            >
              {`拖拽邮箱到此，自动归入默认分组“${defaultGroupName}”`}
            </div>
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

      <Dialog open={groupModalOpen} onOpenChange={setGroupModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{groupModalMode === 'create' ? '新建分组' : '编辑分组'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <Input
              value={groupName}
              onChange={(e) => setGroupName(e.target.value)}
              placeholder="输入分组名称"
              className="h-10"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGroupModalOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSubmitGroupDialog}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
