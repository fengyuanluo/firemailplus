/**
 * 合并后的邮箱状态管理
 * 包含账户、文件夹、邮件和搜索功能
 */

import { create } from 'zustand';
import type { EmailAccount, Email, Folder } from '@/types/email';

// 搜索筛选条件
interface SearchFilters {
  dateRange?: { start: string; end: string };
  hasAttachment?: boolean;
  isRead?: boolean;
  isStarred?: boolean;
  isImportant?: boolean;
  sender?: string;
  subject?: string;
  folder?: number;
  account?: number;
}

// 邮件筛选条件
interface EmailFilters {
  unreadOnly: boolean;
  starredOnly: boolean;
  importantOnly: boolean;
  hasAttachments: boolean;
  dateRange?: { start: string; end: string };
}

// 合并后的邮箱状态
interface MailboxState {
  // 账户状态
  accounts: EmailAccount[];
  selectedAccount: EmailAccount | null;

  // 文件夹状态
  folders: Folder[];
  selectedFolder: Folder | null;
  expandedFolders: Set<number>;

  // 邮件状态
  emails: Email[];
  selectedEmail: Email | null;
  selectedEmails: Set<number>;

  // 分页状态
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;

  // 排序状态
  sortBy: string;
  sortOrder: string;

  // 筛选状态
  activeFilters: EmailFilters;

  // 搜索状态
  searchQuery: string;
  searchFilters: SearchFilters;
  isSearching: boolean;

  // 加载状态
  isLoading: boolean;
  isSyncing: boolean;
  syncError: string | null;

  // 账户操作
  setAccounts: (accounts: EmailAccount[]) => void;
  addAccount: (account: EmailAccount) => void;
  updateAccount: (account: EmailAccount) => void;
  removeAccount: (id: number) => void;
  selectAccount: (account: EmailAccount | null) => void;

  // 文件夹操作
  setFolders: (folders: Folder[]) => void;
  selectFolder: (folder: Folder | null) => void;
  toggleFolderExpansion: (folderId: number) => void;

  // 邮件操作
  setEmails: (emails: Email[]) => void;
  appendEmails: (emails: Email[]) => void;
  addEmail: (email: Email) => void;
  updateEmail: (id: number, updates: Partial<Email>) => void;
  removeEmail: (id: number) => void;
  selectEmail: (email: Email | null) => void;
  toggleEmailSelection: (emailId: number) => void;
  selectAllEmails: () => void;
  clearSelection: () => void;

  // 分页操作
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  setPagination: (pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  }) => void;

  // 排序操作
  setSort: (sortBy: string, sortOrder: string) => void;

  // 筛选操作
  setActiveFilters: (filters: Partial<EmailFilters>) => void;
  clearActiveFilters: () => void;

  // 搜索操作
  setSearchQuery: (query: string) => void;
  setSearchFilters: (filters: Partial<SearchFilters>) => void;
  clearSearch: () => void;
  setSearching: (searching: boolean) => void;

  // 加载状态操作
  setLoading: (loading: boolean) => void;
  setSyncing: (syncing: boolean) => void;
  setSyncError: (error: string | null) => void;

  // 复合操作
  resetState: () => void;
}

export const useMailboxStore = create<MailboxState>((set, get) => ({
  // 初始状态
  accounts: [],
  selectedAccount: null,
  folders: [],
  selectedFolder: null,
  expandedFolders: new Set(),
  emails: [],
  selectedEmail: null,
  selectedEmails: new Set(),
  page: 1,
  pageSize: 20,
  total: 0,
  totalPages: 0,
  sortBy: 'date',
  sortOrder: 'desc',
  activeFilters: {
    unreadOnly: false,
    starredOnly: false,
    importantOnly: false,
    hasAttachments: false,
  },
  searchQuery: '',
  searchFilters: {},
  isSearching: false,
  isLoading: false,
  isSyncing: false,
  syncError: null,

  // 账户操作
  setAccounts: (accounts) => set({ accounts }),
  addAccount: (account) =>
    set((state) => ({
      accounts: [...state.accounts, account],
    })),
  updateAccount: (account) =>
    set((state) => ({
      accounts: state.accounts.map((acc) => (acc.id === account.id ? account : acc)),
      selectedAccount: state.selectedAccount?.id === account.id ? account : state.selectedAccount,
    })),
  removeAccount: (id) =>
    set((state) => ({
      accounts: state.accounts.filter((acc) => acc.id !== id),
      selectedAccount: state.selectedAccount?.id === id ? null : state.selectedAccount,
    })),
  selectAccount: (account) =>
    set({
      selectedAccount: account,
      selectedFolder: null,
      emails: [],
      selectedEmail: null,
      selectedEmails: new Set(),
      page: 1,
    }),

  // 文件夹操作
  setFolders: (folders) => set({ folders }),
  selectFolder: (folder) =>
    set((state) => {
      // 如果选择了文件夹，需要确保有对应的账户被选中
      let selectedAccount = state.selectedAccount;
      if (folder && !selectedAccount) {
        // 如果没有选中账户，自动选择文件夹所属的账户
        selectedAccount = state.accounts.find((acc) => acc.id === folder.account_id) || null;
      }

      return {
        selectedAccount,
        selectedFolder: folder,
        emails: [],
        selectedEmail: null,
        selectedEmails: new Set(),
        page: 1,
      };
    }),
  toggleFolderExpansion: (folderId) =>
    set((state) => {
      const newExpanded = new Set(state.expandedFolders);
      if (newExpanded.has(folderId)) {
        newExpanded.delete(folderId);
      } else {
        newExpanded.add(folderId);
      }
      return { expandedFolders: newExpanded };
    }),

  // 邮件操作
  setEmails: (emails) => set({ emails }),
  appendEmails: (newEmails) =>
    set((state) => {
      // 过滤重复邮件，避免重复添加
      const existingIds = new Set(state.emails.map((email) => email.id));
      const uniqueNewEmails = newEmails.filter((email) => !existingIds.has(email.id));

      return {
        emails: [...state.emails, ...uniqueNewEmails],
      };
    }),
  addEmail: (email) =>
    set((state) => ({
      emails: [email, ...state.emails],
      total: state.total + 1,
    })),
  updateEmail: (id, updates) =>
    set((state) => ({
      emails: state.emails.map((email) => (email.id === id ? { ...email, ...updates } : email)),
      selectedEmail:
        state.selectedEmail?.id === id
          ? { ...state.selectedEmail, ...updates }
          : state.selectedEmail,
    })),
  removeEmail: (id) =>
    set((state) => {
      const newSelectedEmails = new Set(state.selectedEmails);
      newSelectedEmails.delete(id);
      return {
        emails: state.emails.filter((email) => email.id !== id),
        selectedEmail: state.selectedEmail?.id === id ? null : state.selectedEmail,
        selectedEmails: newSelectedEmails,
        total: Math.max(0, state.total - 1),
      };
    }),
  selectEmail: (email) => set({ selectedEmail: email }),
  toggleEmailSelection: (emailId) =>
    set((state) => {
      const newSelectedEmails = new Set(state.selectedEmails);
      if (newSelectedEmails.has(emailId)) {
        newSelectedEmails.delete(emailId);
      } else {
        newSelectedEmails.add(emailId);
      }
      return { selectedEmails: newSelectedEmails };
    }),
  selectAllEmails: () =>
    set((state) => ({
      selectedEmails: new Set(state.emails.map((email) => email.id)),
    })),
  clearSelection: () => set({ selectedEmails: new Set() }),

  // 分页操作
  setPage: (page) => set({ page }),
  setPageSize: (pageSize) => set({ pageSize, page: 1 }),
  setPagination: (pagination) => set(pagination),

  // 排序操作
  setSort: (sortBy, sortOrder) => set({ sortBy, sortOrder, page: 1 }),

  // 筛选操作
  setActiveFilters: (filters) =>
    set((state) => ({
      activeFilters: { ...state.activeFilters, ...filters },
      page: 1,
    })),
  clearActiveFilters: () =>
    set({
      activeFilters: {
        unreadOnly: false,
        starredOnly: false,
        importantOnly: false,
        hasAttachments: false,
      },
      page: 1,
    }),

  // 搜索操作
  setSearchQuery: (query) =>
    set({
      searchQuery: query,
      selectedEmails: new Set(),
      selectedEmail: null,
      page: 1,
    }),
  setSearchFilters: (filters) =>
    set((state) => ({
      searchFilters: { ...state.searchFilters, ...filters },
    })),
  clearSearch: () =>
    set({
      searchQuery: '',
      searchFilters: {},
      isSearching: false,
    }),
  setSearching: (searching) => set({ isSearching: searching }),

  // 加载状态操作
  setLoading: (loading) => set({ isLoading: loading }),
  setSyncing: (syncing) => set({ isSyncing: syncing }),
  setSyncError: (error) => set({ syncError: error }),

  // 复合操作
  resetState: () =>
    set({
      emails: [],
      selectedEmail: null,
      selectedEmails: new Set(),
      page: 1,
      isLoading: false,
      isSyncing: false,
      syncError: null,
    }),
}));
