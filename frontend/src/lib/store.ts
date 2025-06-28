/**
 * Zustand 状态管理
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { LoginResponse, User } from './api';
import type { EmailAccount, Email, Folder } from '@/types/email';
import { parseEmailAddress, parseEmailAddresses } from '@/types/email';

// 认证状态
interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isHydrated: boolean;
  login: (user: User, token: string) => void;
  logout: () => void;
  setHydrated: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isHydrated: false,
      login: (user, token) => {
        set({ user, token, isAuthenticated: true });
      },
      logout: () => {
        set({ user: null, token: null, isAuthenticated: false });
      },
      setHydrated: () => {
        set({ isHydrated: true });
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        // 水合完成后设置标志
        if (state) {
          state.setHydrated();
        }
      },
    }
  )
);

// 邮箱相关状态已移至 mailbox-store.ts
export { useMailboxStore } from './mailbox-store';

// UI 状态
interface UIState {
  sidebarOpen: boolean;
  sidebarOpenMobile: boolean; // 移动端独立的侧边栏状态
  isMobile: boolean;
  theme: 'light' | 'dark' | 'system';

  // 布局状态
  emailListWidth: number;
  emailDetailWidth: number;
  isResizing: boolean;

  // 视图状态
  emailListView: 'compact' | 'comfortable' | 'spacious';
  showPreview: boolean;
  showAttachments: boolean;

  // 加载状态
  isInitializing: boolean;
  isRefreshing: boolean;

  // 键盘快捷键状态
  keyboardShortcutsEnabled: boolean;
  showKeyboardHelp: boolean;

  // 基础操作
  setSidebarOpen: (open: boolean) => void;
  setSidebarOpenMobile: (open: boolean) => void;
  setIsMobile: (mobile: boolean) => void;
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  toggleSidebar: () => void;

  // 布局操作
  setEmailListWidth: (width: number) => void;
  setEmailDetailWidth: (width: number) => void;
  setResizing: (resizing: boolean) => void;
  resetLayout: () => void;

  // 视图操作
  setEmailListView: (view: 'compact' | 'comfortable' | 'spacious') => void;
  setShowPreview: (show: boolean) => void;
  setShowAttachments: (show: boolean) => void;
  togglePreview: () => void;

  // 加载操作
  setInitializing: (initializing: boolean) => void;
  setRefreshing: (refreshing: boolean) => void;

  // 键盘快捷键操作
  setKeyboardShortcutsEnabled: (enabled: boolean) => void;
  setShowKeyboardHelp: (show: boolean) => void;
  toggleKeyboardHelp: () => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      sidebarOpen: true, // 桌面端默认打开
      sidebarOpenMobile: false, // 移动端默认关闭
      isMobile: false,
      theme: 'system',

      // 布局状态
      emailListWidth: 400,
      emailDetailWidth: 600,
      isResizing: false,

      // 视图状态
      emailListView: 'comfortable',
      showPreview: true,
      showAttachments: true,

      // 加载状态
      isInitializing: false,
      isRefreshing: false,

      // 键盘快捷键状态
      keyboardShortcutsEnabled: true,
      showKeyboardHelp: false,

      // 基础操作
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSidebarOpenMobile: (open) => set({ sidebarOpenMobile: open }),
      setIsMobile: (mobile) => set({ isMobile: mobile }),
      setTheme: (theme) => set({ theme }),
      toggleSidebar: () =>
        set((state) => {
          const { isMobile } = get();
          if (isMobile) {
            return { sidebarOpenMobile: !state.sidebarOpenMobile };
          } else {
            return { sidebarOpen: !state.sidebarOpen };
          }
        }),

      // 布局操作
      setEmailListWidth: (width) => set({ emailListWidth: Math.max(300, Math.min(600, width)) }),
      setEmailDetailWidth: (width) => set({ emailDetailWidth: Math.max(400, width) }),
      setResizing: (resizing) => set({ isResizing: resizing }),
      resetLayout: () =>
        set({
          emailListWidth: 400,
          emailDetailWidth: 600,
          sidebarOpen: true,
          sidebarOpenMobile: false,
        }),

      // 视图操作
      setEmailListView: (view) => set({ emailListView: view }),
      setShowPreview: (show) => set({ showPreview: show }),
      setShowAttachments: (show) => set({ showAttachments: show }),
      togglePreview: () => set((state) => ({ showPreview: !state.showPreview })),

      // 加载操作
      setInitializing: (initializing) => set({ isInitializing: initializing }),
      setRefreshing: (refreshing) => set({ isRefreshing: refreshing }),

      // 键盘快捷键操作
      setKeyboardShortcutsEnabled: (enabled) => set({ keyboardShortcutsEnabled: enabled }),
      setShowKeyboardHelp: (show) => set({ showKeyboardHelp: show }),
      toggleKeyboardHelp: () => set((state) => ({ showKeyboardHelp: !state.showKeyboardHelp })),
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({
        sidebarOpen: state.sidebarOpen,
        sidebarOpenMobile: state.sidebarOpenMobile,
        theme: state.theme,
        emailListWidth: state.emailListWidth,
        emailDetailWidth: state.emailDetailWidth,
        emailListView: state.emailListView,
        showPreview: state.showPreview,
        showAttachments: state.showAttachments,
        keyboardShortcutsEnabled: state.keyboardShortcutsEnabled,
      }),
    }
  )
);

// 收件人接口
interface Recipient {
  email: string;
  name?: string;
  isValid: boolean;
}

// 附件文件接口
interface AttachmentFile {
  id: string;
  file: File;
  name: string;
  size: number;
  type: string;
  uploadProgress: number;
  uploadStatus: 'pending' | 'uploading' | 'completed' | 'error';
  attachmentId?: number;
  errorMessage?: string;
}

// 发送选项接口
interface SendOptions {
  priority: 'low' | 'normal' | 'high';
  scheduledTime?: string;
  requestReadReceipt: boolean;
  requestDeliveryReceipt: boolean;
  importance: 'low' | 'normal' | 'high';
}

// 写信状态
interface ComposeState {
  isOpen: boolean;
  mode: 'compose' | 'reply' | 'replyAll' | 'forward';
  originalEmailId: number | null;
  draft: {
    accountId?: number;
    to: Recipient[];
    cc: Recipient[];
    bcc: Recipient[];
    subject: string;
    content: string;
    htmlContent: string;
    attachments: AttachmentFile[];
    templateId?: number;
    sendOptions: SendOptions;
  };
  autoSaveStatus: 'idle' | 'saving' | 'saved' | 'error';
  sendStatus: 'idle' | 'sending' | 'sent' | 'failed';
  setIsOpen: (open: boolean) => void;
  updateDraft: (updates: Partial<ComposeState['draft']>) => void;
  updateContent: (html: string, text: string) => void;
  updateRecipients: (type: 'to' | 'cc' | 'bcc', recipients: Recipient[]) => void;
  updateAttachments: (
    attachments: AttachmentFile[] | ((prev: AttachmentFile[]) => AttachmentFile[])
  ) => void;
  updateSendOptions: (options: SendOptions) => void;
  clearDraft: () => void;
  setAutoSaveStatus: (status: ComposeState['autoSaveStatus']) => void;
  setSendStatus: (status: ComposeState['sendStatus']) => void;
  initializeReply: (originalEmail: any) => void;
  initializeReplyAll: (originalEmail: any) => void;
  initializeForward: (originalEmail: any) => void;
  openCompose: () => void;
}

export const useComposeStore = create<ComposeState>()(
  persist(
    (set) => ({
      isOpen: false,
      mode: 'compose',
      originalEmailId: null,
      draft: {
        to: [],
        cc: [],
        bcc: [],
        subject: '',
        content: '',
        htmlContent: '',
        attachments: [],
        sendOptions: {
          priority: 'normal',
          requestReadReceipt: false,
          requestDeliveryReceipt: false,
          importance: 'normal',
        },
      },
      autoSaveStatus: 'idle',
      sendStatus: 'idle',
      setIsOpen: (open) => set({ isOpen: open }),
      updateDraft: (updates) =>
        set((state) => ({
          draft: { ...state.draft, ...updates },
        })),
      updateContent: (html, text) =>
        set((state) => ({
          draft: {
            ...state.draft,
            htmlContent: html,
            content: text,
          },
        })),
      updateRecipients: (type, recipients) =>
        set((state) => ({
          draft: {
            ...state.draft,
            [type]: recipients,
          },
        })),
      updateAttachments: (attachments) =>
        set((state) => ({
          draft: {
            ...state.draft,
            attachments:
              typeof attachments === 'function'
                ? attachments(state.draft.attachments)
                : attachments,
          },
        })),
      updateSendOptions: (options) =>
        set((state) => ({
          draft: {
            ...state.draft,
            sendOptions: options,
          },
        })),
      clearDraft: () =>
        set({
          draft: {
            to: [],
            cc: [],
            bcc: [],
            subject: '',
            content: '',
            htmlContent: '',
            attachments: [],
            sendOptions: {
              priority: 'normal',
              requestReadReceipt: false,
              requestDeliveryReceipt: false,
              importance: 'normal',
            },
          },
          autoSaveStatus: 'idle',
          sendStatus: 'idle',
        }),
      setAutoSaveStatus: (status) => set({ autoSaveStatus: status }),
      setSendStatus: (status) => set({ sendStatus: status }),
      initializeReply: (originalEmail) => {
        // 解析原邮件的发件人作为回复的收件人
        const fromAddress = parseEmailAddress(originalEmail.from);

        const replyTo = fromAddress
          ? [
              {
                email: fromAddress.address,
                name: fromAddress.name || '',
                isValid: true,
              },
            ]
          : [];

        // 构建回复主题
        const replySubject = originalEmail.subject.startsWith('Re:')
          ? originalEmail.subject
          : `Re: ${originalEmail.subject}`;

        set({
          isOpen: true,
          mode: 'reply',
          originalEmailId: originalEmail.id,
          draft: {
            to: replyTo,
            cc: [],
            bcc: [],
            subject: replySubject,
            content: '',
            htmlContent: '',
            attachments: [],
            sendOptions: {
              priority: 'normal',
              requestReadReceipt: false,
              requestDeliveryReceipt: false,
              importance: 'normal',
            },
          },
        });
      },
      initializeReplyAll: (originalEmail) => {
        // 解析原邮件的发件人
        const fromAddress = parseEmailAddress(originalEmail.from);

        // 解析原邮件的收件人和抄送人
        const toAddresses = parseEmailAddresses(originalEmail.to || '');
        const ccAddresses = parseEmailAddresses(originalEmail.cc || '');

        // 构建回复收件人列表（包含原发件人）
        const replyTo = fromAddress
          ? [
              {
                email: fromAddress.address,
                name: fromAddress.name || '',
                isValid: true,
              },
            ]
          : [];

        // 构建回复抄送列表（包含原收件人和抄送人，但排除自己）
        const replyCc = [...toAddresses, ...ccAddresses]
          .filter((addr) => addr.address !== fromAddress?.address) // 排除原发件人
          .map((addr) => ({
            email: addr.address,
            name: addr.name || '',
            isValid: true,
          }));

        // 构建回复主题
        const replySubject = originalEmail.subject.startsWith('Re:')
          ? originalEmail.subject
          : `Re: ${originalEmail.subject}`;

        set({
          isOpen: true,
          mode: 'replyAll',
          originalEmailId: originalEmail.id,
          draft: {
            to: replyTo,
            cc: replyCc,
            bcc: [],
            subject: replySubject,
            content: '',
            htmlContent: '',
            attachments: [],
            sendOptions: {
              priority: 'normal',
              requestReadReceipt: false,
              requestDeliveryReceipt: false,
              importance: 'normal',
            },
          },
        });
      },
      initializeForward: (originalEmail) => {
        // 构建转发主题
        const forwardSubject = originalEmail.subject.startsWith('Fwd:')
          ? originalEmail.subject
          : `Fwd: ${originalEmail.subject}`;

        // 解析邮件地址用于显示
        const fromAddress = parseEmailAddress(originalEmail.from);
        const toAddresses = parseEmailAddresses(originalEmail.to || '');

        const fromDisplay = fromAddress
          ? fromAddress.name
            ? `${fromAddress.name} <${fromAddress.address}>`
            : fromAddress.address
          : originalEmail.from;
        const toDisplay =
          toAddresses.length > 0
            ? toAddresses
                .map((addr) => (addr.name ? `${addr.name} <${addr.address}>` : addr.address))
                .join(', ')
            : originalEmail.to;

        // 构建转发内容（包含原邮件信息）
        const originalContent = originalEmail.html_body || originalEmail.text_body || '';
        const forwardContent = `

---------- 转发邮件 ----------
发件人: ${fromDisplay}
收件人: ${toDisplay}
发送时间: ${originalEmail.date}
主题: ${originalEmail.subject}

${originalContent}`;

        set({
          isOpen: true,
          mode: 'forward',
          originalEmailId: originalEmail.id,
          draft: {
            to: [],
            cc: [],
            bcc: [],
            subject: forwardSubject,
            content: forwardContent,
            htmlContent: forwardContent,
            attachments: [],
            sendOptions: {
              priority: 'normal',
              requestReadReceipt: false,
              requestDeliveryReceipt: false,
              importance: 'normal',
            },
          },
        });
      },
      openCompose: () =>
        set({
          isOpen: true,
          mode: 'compose',
          originalEmailId: null,
          // 重置草稿内容，确保不会保留之前的状态
          draft: {
            to: [],
            cc: [],
            bcc: [],
            subject: '',
            content: '',
            htmlContent: '',
            attachments: [],
            sendOptions: {
              priority: 'normal',
              requestReadReceipt: false,
              requestDeliveryReceipt: false,
              importance: 'normal',
            },
          },
        }),
    }),
    {
      name: 'compose-storage',
      partialize: (state) => ({
        draft: state.draft,
        mode: state.mode,
        originalEmailId: state.originalEmailId,
      }),
    }
  )
);

// 搜索相关状态已移至 mailbox-store.ts

// 导航状态
interface NavigationState {
  currentView: 'inbox' | 'sent' | 'drafts' | 'trash' | 'search' | 'folder' | 'all';
  breadcrumbs: Array<{ label: string; path: string; id?: number }>;
  canGoBack: boolean;
  canGoForward: boolean;
  history: string[];
  historyIndex: number;

  // 基础操作
  setCurrentView: (view: NavigationState['currentView']) => void;
  setBreadcrumbs: (breadcrumbs: NavigationState['breadcrumbs']) => void;
  addToBreadcrumbs: (item: { label: string; path: string; id?: number }) => void;
  goBack: () => void;
  goForward: () => void;
  navigateTo: (path: string) => void;
  clearHistory: () => void;
}

// 通知接口
interface Notification {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  autoClose?: boolean;
  duration?: number;
}

// 通知状态
interface NotificationState {
  notifications: Notification[];
  unreadCount: number;

  // 基础操作
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => void;
  removeNotification: (id: string) => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  clearAll: () => void;
  clearRead: () => void;
}

// 右键菜单状态
interface ContextMenuState {
  isOpen: boolean;
  position: { x: number; y: number };
  target: {
    type: 'account' | 'folder' | 'email';
    id: number;
    data?: any;
  } | null;
  setIsOpen: (open: boolean) => void;
  setPosition: (position: { x: number; y: number }) => void;
  setTarget: (target: ContextMenuState['target']) => void;
  openMenu: (position: { x: number; y: number }, target: ContextMenuState['target']) => void;
  closeMenu: () => void;
}

// 导航状态实现
export const useNavigationStore = create<NavigationState>((set, get) => ({
  currentView: 'inbox',
  breadcrumbs: [{ label: '收件箱', path: '/mailbox' }],
  canGoBack: false,
  canGoForward: false,
  history: ['/mailbox'],
  historyIndex: 0,

  // 基础操作
  setCurrentView: (view) => set({ currentView: view }),
  setBreadcrumbs: (breadcrumbs) => set({ breadcrumbs }),
  addToBreadcrumbs: (item) =>
    set((state) => ({
      breadcrumbs: [...state.breadcrumbs, item],
    })),
  goBack: () =>
    set((state) => {
      if (state.historyIndex > 0) {
        return {
          historyIndex: state.historyIndex - 1,
          canGoBack: state.historyIndex - 1 > 0,
          canGoForward: true,
        };
      }
      return state;
    }),
  goForward: () =>
    set((state) => {
      if (state.historyIndex < state.history.length - 1) {
        return {
          historyIndex: state.historyIndex + 1,
          canGoBack: true,
          canGoForward: state.historyIndex + 1 < state.history.length - 1,
        };
      }
      return state;
    }),
  navigateTo: (path) =>
    set((state) => {
      const newHistory = state.history.slice(0, state.historyIndex + 1);
      newHistory.push(path);
      return {
        history: newHistory,
        historyIndex: newHistory.length - 1,
        canGoBack: newHistory.length > 1,
        canGoForward: false,
      };
    }),
  clearHistory: () =>
    set({
      history: ['/mailbox'],
      historyIndex: 0,
      canGoBack: false,
      canGoForward: false,
    }),
}));

// 通知状态实现
export const useNotificationStore = create<NotificationState>((set, get) => ({
  notifications: [],
  unreadCount: 0,

  // 基础操作
  addNotification: (notification) =>
    set((state) => {
      const newNotification: Notification = {
        ...notification,
        id: Date.now().toString() + Math.random().toString(36).substr(2, 9),
        timestamp: Date.now(),
        read: false,
      };

      return {
        notifications: [newNotification, ...state.notifications],
        unreadCount: state.unreadCount + 1,
      };
    }),
  removeNotification: (id) =>
    set((state) => {
      const notification = state.notifications.find((n) => n.id === id);
      return {
        notifications: state.notifications.filter((n) => n.id !== id),
        unreadCount: notification && !notification.read ? state.unreadCount - 1 : state.unreadCount,
      };
    }),
  markAsRead: (id) =>
    set((state) => {
      const notifications = state.notifications.map((n) =>
        n.id === id ? { ...n, read: true } : n
      );
      const wasUnread = state.notifications.find((n) => n.id === id && !n.read);
      return {
        notifications,
        unreadCount: wasUnread ? state.unreadCount - 1 : state.unreadCount,
      };
    }),
  markAllAsRead: () =>
    set((state) => ({
      notifications: state.notifications.map((n) => ({ ...n, read: true })),
      unreadCount: 0,
    })),
  clearAll: () =>
    set({
      notifications: [],
      unreadCount: 0,
    }),
  clearRead: () =>
    set((state) => ({
      notifications: state.notifications.filter((n) => !n.read),
      unreadCount: state.unreadCount,
    })),
}));

export const useContextMenuStore = create<ContextMenuState>((set) => ({
  isOpen: false,
  position: { x: 0, y: 0 },
  target: null,
  setIsOpen: (open) => set({ isOpen: open }),
  setPosition: (position) => set({ position }),
  setTarget: (target) => set({ target }),
  openMenu: (position, target) =>
    set({
      isOpen: true,
      position,
      target,
    }),
  closeMenu: () =>
    set({
      isOpen: false,
      target: null,
    }),
}));
