'use client';

import { useEffect, useRef } from 'react';
import { CheckCheck, RefreshCw, Settings, Trash2, Edit, FolderPlus, Star } from 'lucide-react';
import { useContextMenuStore, useMailboxStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import type { EmailAccount, Folder, Email } from '@/types/email';

interface ContextMenuProps {
  onMarkAllAsRead?: (targetId: number) => void;
  onSync?: (targetId: number) => void;
  onDelete?: (targetId: number) => void;
  onRename?: (targetId: number) => void;
  onSettings?: (targetId: number) => void;
  onDeleteAccount?: (targetId: number) => void;
  onCreateGroup?: () => void;
  onEditGroup?: (targetId: number) => void;
  onDeleteGroup?: (targetId: number) => void;
  onSetDefaultGroup?: (targetId: number) => void;
}

export function ContextMenu({
  onMarkAllAsRead,
  onSync,
  onDelete,
  onRename,
  onSettings,
  onDeleteAccount,
  onCreateGroup,
  onEditGroup,
  onDeleteGroup,
  onSetDefaultGroup,
}: ContextMenuProps) {
  const { isOpen, position, target, closeMenu } = useContextMenuStore();
  const { accounts, folders, setFolders, setEmails, updateAccount, emails } = useMailboxStore();
  const menuRef = useRef<HTMLDivElement>(null);

  // 点击外部关闭菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        closeMenu();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closeMenu();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, closeMenu]);

  // 处理菜单项点击
  const handleMenuItemClick = async (action: string) => {
    if (!target) return;

    try {
      switch (action) {
        case 'markAccountAsRead':
          if (target.type === 'account') {
            await apiClient.markAccountAsRead(target.id);
            toast.success('已标记该邮箱所有邮件为已读');

            const account = accounts.find((acc) => acc.id === target.id);
            if (account) {
              const updatedAccount: EmailAccount = { ...account, unread_emails: 0 };
              updateAccount(updatedAccount);
            }

            // 将该账户的文件夹未读清零
            setFolders(
              folders.map((f) =>
                f.account_id === target.id ? { ...f, unread_emails: 0 } : f
              )
            );

            // 将当前列表中属于该账户的邮件标记为已读
            if (emails.length > 0) {
              const updatedEmails: Email[] = emails.map((mail) =>
                mail.account_id === target.id ? { ...mail, is_read: true } : mail
              );
              setEmails(updatedEmails);
            }
          }
          break;

        case 'markAllAsRead':
          if (target.type === 'folder') {
            await apiClient.markFolderAsRead(target.id);
            toast.success('已标记文件夹内所有邮件为已读');
            onMarkAllAsRead?.(target.id);

            // 本地同步未读计数和邮件状态
            const folderData = target.data as Folder | undefined;
            const unreadDelta = folderData?.unread_emails ?? 0;

            if (folderData) {
              // 更新文件夹未读为 0
              setFolders(
                folders.map((f) => (f.id === folderData.id ? { ...f, unread_emails: 0 } : f))
              );

              // 更新账户未读计数
              if (unreadDelta > 0) {
                const account = accounts.find((acc) => acc.id === folderData.account_id);
                if (account) {
                  const nextCount = Math.max(0, account.unread_emails - unreadDelta);
                  const updatedAccount: EmailAccount = { ...account, unread_emails: nextCount };
                  updateAccount(updatedAccount);
                }
              }
            }

            // 将当前列表中该文件夹的邮件标记为已读
            if (emails.length > 0) {
              const updatedEmails: Email[] = emails.map((mail) =>
                mail.folder_id === target.id ? { ...mail, is_read: true } : mail
              );
              setEmails(updatedEmails);
            }
          }
          break;

        case 'sync':
          if (target.type === 'account') {
            await apiClient.syncAccount(target.id);
            toast.success('账户同步已开始');
          } else if (target.type === 'folder') {
            await apiClient.syncFolder(target.id);
            toast.success('文件夹同步已开始');
          }
          onSync?.(target.id);
          break;

        case 'delete':
          if (target.type === 'folder' && target.data?.type === 'custom') {
            if (confirm('确定要删除此文件夹吗？此操作不可撤销。')) {
              await apiClient.deleteFolder(target.id);
              toast.success('文件夹已删除');
              onDelete?.(target.id);
            }
          }
          break;

        case 'rename':
          if (target.type === 'folder' && target.data?.type === 'custom') {
            onRename?.(target.id);
          }
          break;

        case 'settings':
          if (target.type === 'account') {
            onSettings?.(target.id);
          }
          break;

        case 'deleteAccount':
          if (target.type === 'account') {
            onDeleteAccount?.(target.id);
          }
          break;

        case 'createGroup':
          onCreateGroup?.();
          break;

        case 'editGroup':
          if (target.type === 'group') {
            onEditGroup?.(target.id);
          }
          break;

        case 'deleteGroup':
          if (target.type === 'group') {
            onDeleteGroup?.(target.id);
          }
          break;

        case 'setDefaultGroup':
          if (target.type === 'group' && !target.data?.is_default) {
            onSetDefaultGroup?.(target.id);
          }
          break;
      }
    } catch (error: any) {
      toast.error(error.message || '操作失败');
    }

    closeMenu();
  };

  if (!isOpen || !target || target.type === 'email') return null;

  // 根据目标类型生成菜单项
  const getMenuItems = () => {
    const items = [];

    if (target.type === 'account') {
      items.push(
        {
          icon: CheckCheck,
          label: '全部已读',
          action: 'markAccountAsRead',
        },
        {
          icon: RefreshCw,
          label: '同步邮件',
          action: 'sync',
        },
        {
          icon: Settings,
          label: '账户设置',
          action: 'settings',
        },
        {
          icon: Trash2,
          label: '删除账户',
          action: 'deleteAccount',
          danger: true,
        }
      );
    } else if (target.type === 'group') {
      if (!target.data?.is_default) {
        items.push({
          icon: Star,
          label: '设为默认',
          action: 'setDefaultGroup',
        });
      }
      items.push(
        {
          icon: Edit,
          label: '重命名分组',
          action: 'editGroup',
        },
        {
          icon: Trash2,
          label: '删除分组',
          action: 'deleteGroup',
          danger: true,
        }
      );
    } else if (target.type === 'group-blank') {
      items.push({
        icon: FolderPlus,
        label: '新建分组',
        action: 'createGroup',
      });
    } else if (target.type === 'folder') {
      items.push(
        {
          icon: CheckCheck,
          label: '全部已读',
          action: 'markAllAsRead',
        },
        {
          icon: RefreshCw,
          label: '手动同步',
          action: 'sync',
        }
      );

      // 自定义文件夹才能重命名和删除
      if (target.data?.type === 'custom') {
        items.push(
          {
            icon: Edit,
            label: '重命名',
            action: 'rename',
          },
          {
            icon: Trash2,
            label: '删除文件夹',
            action: 'delete',
            danger: true,
          }
        );
      }
    }

    return items;
  };

  const menuItems = getMenuItems();

  return (
    <div
      ref={menuRef}
      className="fixed z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg py-1 min-w-[160px]"
      style={{
        left: position.x,
        top: position.y,
      }}
    >
      {menuItems.map((item, index) => (
        <button
          key={index}
          onClick={() => handleMenuItemClick(item.action)}
          className={`
            w-full flex items-center gap-2 px-3 py-2 text-sm text-left
            hover:bg-gray-100 dark:hover:bg-gray-700
            ${
              item.danger
                ? 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20'
                : 'text-gray-700 dark:text-gray-300'
            }
          `}
        >
          <item.icon className="w-4 h-4" />
          {item.label}
        </button>
      ))}
    </div>
  );
}
