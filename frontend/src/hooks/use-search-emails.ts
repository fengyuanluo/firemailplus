/**
 * 邮件搜索专用 Hook
 */

import { useState, useCallback, useEffect, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import type { Email } from '@/types/email';

// 搜索参数接口 - 与后端API保持一致
export interface SearchParams {
  q?: string; // 全文搜索关键词
  subject?: string; // 主题搜索
  from?: string; // 发件人搜索
  to?: string; // 收件人搜索
  body?: string; // 正文搜索
  account_id?: number; // 账户ID筛选
  folder_id?: number; // 文件夹ID筛选
  has_attachment?: boolean; // 是否有附件
  is_read?: boolean; // 是否已读
  is_starred?: boolean; // 是否加星
  is_important?: boolean; // 是否重要（前端扩展，后端可能不支持）
  since?: string; // 开始时间 (RFC3339格式)
  before?: string; // 结束时间 (RFC3339格式)
  page?: number; // 页码
  page_size?: number; // 每页大小
}

// 搜索结果接口
export interface SearchResult {
  emails: Email[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useSearchEmails() {
  const { selectEmail } = useMailboxStore();

  // 搜索状态
  const [searchParams, setSearchParams] = useState<SearchParams>({
    page: 1,
    page_size: 20,
  });

  const [selectedEmailId, setSelectedEmailId] = useState<number | null>(null);
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);
  const pageSizeRef = useRef(searchParams.page_size);

  // 保持pageSizeRef与当前值同步
  pageSizeRef.current = searchParams.page_size;

  // 构建查询键
  const queryKey = ['searchEmails', searchParams];

  // 搜索API调用
  const { data, isLoading, error, refetch } = useQuery({
    queryKey,
    queryFn: async () => {
      console.log('🔍 [useSearchEmails] 开始搜索请求:', {
        searchParams,
        queryKey,
        timestamp: new Date().toISOString(),
      });

      if (!searchParams.q?.trim()) {
        console.log('🔍 [useSearchEmails] 搜索查询为空，返回空结果');
        return { emails: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
      }

      try {
        console.log('🔍 [useSearchEmails] 调用 API 搜索邮件:', searchParams);
        const response = await apiClient.searchEmails(searchParams);

        console.log('🔍 [useSearchEmails] API 响应:', {
          success: response.success,
          dataExists: !!response.data,
          emailCount: response.data?.emails?.length || 0,
          total: response.data?.total || 0,
          message: response.message,
        });

        if (response.success && response.data) {
          console.log('🔍 [useSearchEmails] 搜索成功，返回数据:', {
            emails: response.data.emails.map((email) => ({
              id: email.id,
              subject: email.subject,
              from: email.from,
              date: email.date,
            })),
            total: response.data.total,
            page: response.data.page,
            page_size: response.data.page_size,
            total_pages: response.data.total_pages,
          });
          return response.data;
        }

        console.error('🔍 [useSearchEmails] 搜索失败:', response.message);
        throw new Error(response.message || '搜索失败');
      } catch (error) {
        console.error('🔍 [useSearchEmails] 搜索异常:', error);
        throw error;
      }
    },
    enabled: !!searchParams.q?.trim(),
    staleTime: 30000, // 30秒内不重新请求
    gcTime: 300000, // 5分钟后清理缓存
  });

  // 简单防抖实现
  const debouncedSearch = useCallback((params: SearchParams) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    debounceTimerRef.current = setTimeout(() => {
      setSearchParams((prev) => ({ ...prev, ...params, page: 1 }));
    }, 300);
  }, []);

  // 搜索方法
  const search = useCallback(
    (query: string, filters: Partial<SearchParams> = {}) => {
      if (process.env.NODE_ENV === 'development') {
        console.log('🔍 [useSearchEmails] search() 被调用:', { query, filters });
      }
      const params = {
        q: query,
        ...filters,
        page: 1,
        page_size: pageSizeRef.current, // 使用ref避免依赖循环
      };
      if (process.env.NODE_ENV === 'development') {
        console.log('🔍 [useSearchEmails] 准备执行防抖搜索:', params);
      }
      debouncedSearch(params);
    },
    [debouncedSearch]
  ); // 移除searchParams.page_size依赖

  // 更新筛选条件
  const updateFilters = useCallback((filters: Partial<SearchParams>) => {
    if (process.env.NODE_ENV === 'development') {
      console.log('🔍 [useSearchEmails] updateFilters() 被调用:', filters);
      console.log('🔍 [useSearchEmails] 当前搜索参数:', searchParams);
    }
    setSearchParams((prev) => {
      const newParams = { ...prev, ...filters, page: 1 };
      if (process.env.NODE_ENV === 'development') {
        console.log('🔍 [useSearchEmails] 更新后的搜索参数:', newParams);
      }
      return newParams;
    });
  }, []);

  // 分页
  const changePage = useCallback((page: number) => {
    setSearchParams((prev) => ({ ...prev, page }));
  }, []);

  // 选择邮件
  const selectSearchEmail = useCallback(
    (emailId: number) => {
      console.log('🔍 [useSearchEmails] selectSearchEmail() 被调用:', emailId);
      setSelectedEmailId(emailId);

      // 同步到全局store，以便EmailDetail组件使用
      const email = data?.emails.find((e) => e.id === emailId);
      console.log(
        '🔍 [useSearchEmails] 找到的邮件:',
        email
          ? {
              id: email.id,
              subject: email.subject,
              from: email.from,
            }
          : null
      );

      if (email) {
        selectEmail(email);
        console.log('🔍 [useSearchEmails] 邮件已同步到全局store');
      }
    },
    [data?.emails, selectEmail]
  );

  // 获取选中的邮件
  const selectedEmail = data?.emails.find((email) => email.id === selectedEmailId) || null;

  // 清除搜索
  const clearSearch = useCallback(() => {
    setSearchParams({ page: 1, page_size: 20 });
    setSelectedEmailId(null);
  }, []);

  return {
    // 搜索状态
    searchParams,
    isLoading,
    error,

    // 搜索结果
    emails: data?.emails || [],
    total: data?.total || 0,
    page: data?.page || 1,
    pageSize: data?.page_size || 20,
    totalPages: data?.total_pages || 0,

    // 选中状态
    selectedEmailId,
    selectedEmail,

    // 搜索方法
    search,
    updateFilters,
    changePage,
    selectSearchEmail,
    clearSearch,
    refetch,
  };
}
