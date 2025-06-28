/**
 * 键盘快捷键管理Hook
 */

import { useEffect, useCallback } from 'react';
import { useMailboxStore } from '@/lib/store';
import { useComposeStore } from '@/lib/store';

// 快捷键配置
const SHORTCUTS = {
  // 导航快捷键
  j: 'next_email', // 下一封邮件
  k: 'previous_email', // 上一封邮件
  o: 'open_email', // 打开邮件
  u: 'back_to_list', // 返回列表

  // 邮件操作快捷键
  r: 'reply', // 回复
  a: 'reply_all', // 回复全部
  f: 'forward', // 转发
  d: 'delete', // 删除
  e: 'archive', // 归档
  s: 'star', // 星标
  i: 'important', // 重要
  m: 'mark_read', // 标记已读
  'shift+m': 'mark_unread', // 标记未读

  // 选择快捷键
  x: 'toggle_select', // 切换选择
  'shift+a': 'select_all', // 全选
  'shift+n': 'select_none', // 取消全选

  // 搜索快捷键
  '/': 'search', // 搜索
  escape: 'clear_search', // 清除搜索

  // 写信快捷键
  c: 'compose', // 写信
  'ctrl+enter': 'send', // 发送邮件
  'ctrl+s': 'save_draft', // 保存草稿

  // 视图快捷键
  'g+i': 'goto_inbox', // 转到收件箱
  'g+s': 'goto_sent', // 转到发件箱
  'g+d': 'goto_drafts', // 转到草稿箱
  'g+t': 'goto_trash', // 转到垃圾箱

  // UI快捷键
  b: 'toggle_sidebar', // 切换侧边栏
  p: 'toggle_preview', // 切换预览
  '?': 'show_help', // 显示帮助
};

export function useKeyboardShortcuts() {
  const mailboxState = useMailboxStore();
  const composeStore = useComposeStore();

  // 处理快捷键
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      // 如果在输入框中，不处理快捷键
      const target = event.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.contentEditable === 'true'
      ) {
        // 只处理特定的快捷键
        if (event.key === 'Escape') {
          target.blur();
          return;
        }
        if (event.ctrlKey && event.key === 'Enter' && composeStore.isOpen) {
          event.preventDefault();
          handleShortcut('send');
          return;
        }
        if (event.ctrlKey && event.key === 's' && composeStore.isOpen) {
          event.preventDefault();
          handleShortcut('save_draft');
          return;
        }
        return;
      }

      // 构建快捷键字符串
      let shortcut = '';
      if (event.ctrlKey) shortcut += 'ctrl+';
      if (event.shiftKey) shortcut += 'shift+';
      if (event.altKey) shortcut += 'alt+';
      shortcut += event.key.toLowerCase();

      // 检查是否是已定义的快捷键
      const action = SHORTCUTS[shortcut as keyof typeof SHORTCUTS];
      if (action) {
        event.preventDefault();
        handleShortcut(action);
      }
    },
    [mailboxState, composeStore.isOpen]
  );

  // 处理快捷键动作
  const handleShortcut = useCallback(
    (action: string) => {
      const { emails, selectedEmail, selectEmail } = mailboxState;

      switch (action) {
        case 'next_email':
          if (emails.length > 0) {
            const currentIndex = selectedEmail
              ? emails.findIndex((e) => e.id === selectedEmail.id)
              : -1;
            const nextIndex = Math.min(currentIndex + 1, emails.length - 1);
            selectEmail(emails[nextIndex]);
          }
          break;

        case 'previous_email':
          if (emails.length > 0) {
            const currentIndex = selectedEmail
              ? emails.findIndex((e) => e.id === selectedEmail.id)
              : -1;
            const prevIndex = Math.max(currentIndex - 1, 0);
            selectEmail(emails[prevIndex]);
          }
          break;

        case 'open_email':
          if (selectedEmail) {
            console.log('打开邮件:', selectedEmail.subject);
          }
          break;

        case 'reply':
        case 'reply_all':
        case 'forward':
        case 'delete':
        case 'archive':
        case 'star':
        case 'mark_read':
        case 'mark_unread':
        case 'toggle_select':
        case 'select_all':
        case 'select_none':
          console.log('快捷键功能开发中:', action);
          break;

        case 'search':
          // 聚焦到搜索框
          const searchInput = document.querySelector('input[type="search"]') as HTMLInputElement;
          if (searchInput) {
            searchInput.focus();
          }
          break;

        case 'clear_search':
          console.log('清除搜索');
          break;

        case 'compose':
          composeStore.openCompose();
          break;

        case 'send':
        case 'save_draft':
        case 'goto_inbox':
        case 'goto_sent':
        case 'goto_drafts':
        case 'goto_trash':
        case 'toggle_sidebar':
        case 'toggle_preview':
        case 'show_help':
          console.log('快捷键功能开发中:', action);
          break;

        default:
          console.log('Unknown shortcut action:', action);
      }
    },
    [mailboxState, composeStore]
  );

  // 注册键盘事件监听器
  useEffect(() => {
    // 默认启用快捷键
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown]);

  // 获取快捷键列表
  const getShortcutsList = useCallback(() => {
    return Object.entries(SHORTCUTS).map(([key, action]) => ({
      key,
      action,
      description: getShortcutDescription(action),
    }));
  }, []);

  return {
    shortcuts: SHORTCUTS,
    getShortcutsList,
    handleShortcut,
    enabled: true,
    setEnabled: () => {}, // 占位函数
  };
}

// 获取快捷键描述
function getShortcutDescription(action: string): string {
  const descriptions: Record<string, string> = {
    next_email: '下一封邮件',
    previous_email: '上一封邮件',
    open_email: '打开邮件',
    back_to_list: '返回列表',
    reply: '回复',
    reply_all: '回复全部',
    forward: '转发',
    delete: '删除',
    archive: '归档',
    star: '星标',
    important: '重要',
    mark_read: '标记已读',
    mark_unread: '标记未读',
    toggle_select: '切换选择',
    select_all: '全选',
    select_none: '取消全选',
    search: '搜索',
    clear_search: '清除搜索',
    compose: '写信',
    send: '发送邮件',
    save_draft: '保存草稿',
    goto_inbox: '转到收件箱',
    goto_sent: '转到发件箱',
    goto_drafts: '转到草稿箱',
    goto_trash: '转到垃圾箱',
    toggle_sidebar: '切换侧边栏',
    toggle_preview: '切换预览',
    show_help: '显示帮助',
  };

  return descriptions[action] || action;
}
