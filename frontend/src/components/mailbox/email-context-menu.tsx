'use client';

import { useEffect, useRef } from 'react';
import {
  Reply,
  ReplyAll,
  Forward,
  CheckCheck,
  Star,
  Archive,
  Trash2,
  FolderOpen,
  Copy,
  Flag,
} from 'lucide-react';
import { useContextMenuStore, useMailboxStore, useComposeStore } from '@/lib/store';
import { Email } from '@/types/email';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

export function EmailContextMenu() {
  const { isOpen, position, target, closeMenu } = useContextMenuStore();
  const { patchEmail, updateEmail, removeEmail, folders } = useMailboxStore();
  const { initializeReply, initializeReplyAll, initializeForward } = useComposeStore();
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
    // 添加调试信息
    console.log('📧 [EmailContextMenu] 菜单项被点击:', { action, target });

    if (!target || target.type !== 'email') {
      console.warn('📧 [EmailContextMenu] 无效的目标或目标类型:', { target });
      closeMenu();
      return;
    }

    const email = target.data as Email;

    try {
      switch (action) {
        case 'reply':
          initializeReply(email);
          toast.success('已打开回复窗口');
          break;

        case 'replyAll':
          initializeReplyAll(email);
          toast.success('已打开回复全部窗口');
          break;

        case 'forward':
          initializeForward(email);
          toast.success('已打开转发窗口');
          break;

        case 'markAsRead':
          if (!email.is_read) {
            await apiClient.markEmailAsRead(email.id);
            patchEmail(email.id, { is_read: true });
            toast.success('已标记为已读');
          }
          break;

        case 'markAsUnread':
          if (email.is_read) {
            await apiClient.markEmailAsUnread(email.id);
            patchEmail(email.id, { is_read: false });
            toast.success('已标记为未读');
          }
          break;

        case 'toggleStar':
          await apiClient.toggleEmailStar(email.id);
          updateEmail(email.id, { is_starred: !email.is_starred });
          toast.success(email.is_starred ? '已移除星标' : '已添加星标');
          break;

        case 'archive':
          await apiClient.archiveEmail(email.id);
          removeEmail(email.id);
          toast.success('邮件已归档');
          break;

        case 'delete':
          if (confirm('确定要删除这封邮件吗？')) {
            await apiClient.deleteEmail(email.id);
            removeEmail(email.id);
            toast.success('邮件已删除');
          }
          break;

        case 'move':
          // 简单的文件夹选择实现
          const availableFolders = folders.filter(
            (f) => f.type !== 'trash' && f.id !== email.folder_id
          );
          if (availableFolders.length === 0) {
            toast.error('没有可用的文件夹');
            break;
          }

          const folderOptions = availableFolders
            .map((f) => `${f.id}: ${f.display_name || f.name}`)
            .join('\n');
          const selectedFolderId = prompt(
            `请选择要移动到的文件夹（输入文件夹ID）：\n\n${folderOptions}`
          );

          if (selectedFolderId) {
            const folderId = parseInt(selectedFolderId);
            const targetFolder = availableFolders.find((f) => f.id === folderId);

            if (targetFolder) {
              await apiClient.moveEmail(email.id, folderId);
              removeEmail(email.id);
              toast.success(`邮件已移动到 ${targetFolder.display_name || targetFolder.name}`);
            } else {
              toast.error('无效的文件夹ID');
            }
          }
          break;

        case 'copy':
          // 复制邮件内容到剪贴板
          const content = `主题: ${email.subject}\n发件人: ${email.from}\n时间: ${email.date}\n\n${email.text_body || email.html_body}`;
          await navigator.clipboard.writeText(content);
          toast.success('邮件内容已复制到剪贴板');
          break;

        case 'markImportant':
          await apiClient.toggleEmailImportant(email.id);
          updateEmail(email.id, { is_important: !email.is_important });
          toast.success(email.is_important ? '已取消重要标记' : '已标记为重要');
          break;
      }
    } catch (error: unknown) {
      console.error('📧 [EmailContextMenu] 操作失败:', error);
      const message = error instanceof Error && error.message ? error.message : '操作失败';
      toast.error(message);
    }

    closeMenu();
  };

  if (!isOpen || !target || target.type !== 'email') return null;

  const email = target.data as Email;

  const menuItems = [
    {
      icon: Reply,
      label: '回复',
      action: 'reply',
    },
    {
      icon: ReplyAll,
      label: '回复全部',
      action: 'replyAll',
    },
    {
      icon: Forward,
      label: '转发',
      action: 'forward',
    },
    { divider: true },
    {
      icon: CheckCheck,
      label: email.is_read ? '标记为未读' : '标记为已读',
      action: email.is_read ? 'markAsUnread' : 'markAsRead',
    },
    {
      icon: Star,
      label: email.is_starred ? '移除星标' : '添加星标',
      action: 'toggleStar',
    },
    {
      icon: Flag,
      label: email.is_important ? '取消重要' : '标记重要',
      action: 'markImportant',
    },
    { divider: true },
    {
      icon: FolderOpen,
      label: '移动到...',
      action: 'move',
    },
    {
      icon: Archive,
      label: '归档',
      action: 'archive',
    },
    {
      icon: Copy,
      label: '复制内容',
      action: 'copy',
    },
    { divider: true },
    {
      icon: Trash2,
      label: '删除',
      action: 'delete',
      danger: true,
    },
  ];

  return (
    <div
      ref={menuRef}
      className="fixed z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg py-1 min-w-[160px]"
      style={{
        left: position.x,
        top: position.y,
      }}
    >
      {menuItems.map((item, index) => {
        if ('divider' in item) {
          return <div key={index} className="h-px bg-gray-200 dark:bg-gray-600 my-1" />;
        }

        return (
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
        );
      })}
    </div>
  );
}
