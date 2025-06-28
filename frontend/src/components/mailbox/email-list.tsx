'use client';

import { useEffect, useCallback, useState, useMemo } from 'react';
import { useMailboxStore } from '@/lib/store';
import { useMailboxSSE } from '@/hooks/use-sse';
import { EmailListHeader } from './email-list-header';
import { EmailItem } from './email-item';
import { LoadingSkeleton } from './loading-skeleton';
import { EmailContextMenu } from './email-context-menu';
import { Button } from '@/components/ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { apiClient } from '@/lib/api';
import { useLazyLoad } from '@/hooks/use-performance';
import type { Email } from '@/types/email';

// 邮件列表组件属性
interface EmailListProps {
  emails?: Email[];
  selectedEmailId?: number | null;
  onEmailSelect?: (emailId: number) => void;
  showPagination?: boolean;
  currentPage?: number;
  totalPages?: number;
  onPageChange?: (page: number) => void;
  title?: string;
  totalCount?: number;
  isLoading?: boolean;
}

export function EmailList({
  emails: externalEmails,
  selectedEmailId,
  onEmailSelect,
  showPagination = false,
  currentPage = 1,
  totalPages = 1,
  onPageChange,
  title,
  totalCount,
  isLoading: externalLoading,
}: EmailListProps = {}) {
  const {
    emails,
    selectedFolder,
    searchQuery,
    isLoading,
    page,
    pageSize,
    total,
    totalPages: storeTotalPages,
    sortBy,
    sortOrder,
    selectedAccount,
    setEmails,
    appendEmails,
    setLoading,
    setPagination,
  } = useMailboxStore();

  // SSE 连接，监听新邮件事件
  const { newEmailCount, clearNewEmailCount } = useMailboxSSE();

  // 无限滚动状态
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [currentLoadPage, setCurrentLoadPage] = useState(1);

  // 调试信息：监控EmailList的props和状态（仅开发环境）
  useEffect(() => {
    if (process.env.NODE_ENV === 'development') {
      console.log('📧 [EmailList] 组件状态更新:', {
        externalEmails: externalEmails?.length || 0,
        storeEmails: emails.length,
        selectedEmailId,
        isLoading: externalLoading || isLoading,
        title,
        totalCount,
        showPagination,
        currentPage,
        totalPages,
        timestamp: new Date().toISOString(),
      });
    }
  }, [
    externalEmails?.length,
    emails.length,
    selectedEmailId,
    externalLoading,
    isLoading,
    title,
    totalCount,
    showPagination,
    currentPage,
    totalPages,
  ]);

  // 加载邮件列表 - 支持追加模式
  const loadEmails = useCallback(
    async (loadPage: number = 1, append: boolean = false) => {
      if (!selectedAccount && !selectedFolder) return;

      if (loadPage === 1) {
        setLoading(true);
      } else {
        setIsLoadingMore(true);
      }

      try {
        const params = {
          account_id: selectedAccount?.id,
          folder_id: selectedFolder?.id,
          page: loadPage,
          page_size: pageSize,
          sort_by: sortBy,
          sort_order: sortOrder,
          search: searchQuery || undefined,
        };

        const response = await apiClient.getEmails(params);
        if (response.success && response.data) {
          if (append) {
            appendEmails(response.data.emails);
          } else {
            setEmails(response.data.emails);
            setCurrentLoadPage(1);
          }

          setPagination({
            page: response.data.page,
            pageSize: response.data.page_size,
            total: response.data.total,
            totalPages: Math.ceil(response.data.total / response.data.page_size),
          });
        }
      } catch (error) {
        console.error('Failed to load emails:', error);
      } finally {
        setLoading(false);
        setIsLoadingMore(false);
      }
    },
    [
      selectedAccount,
      selectedFolder,
      pageSize,
      sortBy,
      sortOrder,
      searchQuery,
      setEmails,
      appendEmails,
      setPagination,
      setLoading,
    ]
  );

  // 加载更多邮件的函数
  const loadMoreEmails = useCallback(async () => {
    if (isLoadingMore || !selectedAccount || !selectedFolder) return [];

    const nextPage = currentLoadPage + 1;
    if (nextPage > storeTotalPages) return [];

    setCurrentLoadPage(nextPage);
    await loadEmails(nextPage, true);
    return [];
  }, [
    isLoadingMore,
    selectedAccount,
    selectedFolder,
    currentLoadPage,
    storeTotalPages,
    loadEmails,
  ]);

  // 判断是否还有更多数据
  const hasMore = useMemo(() => {
    return currentLoadPage < storeTotalPages;
  }, [currentLoadPage, storeTotalPages]);

  // 使用懒加载Hook
  const { lastElementRef } = useLazyLoad(loadMoreEmails, hasMore, 100);

  // 监听依赖变化，重新加载邮件（仅在没有外部邮件数据时）
  useEffect(() => {
    if (!externalEmails) {
      loadEmails(1, false);
    }
  }, [externalEmails, selectedAccount, selectedFolder, searchQuery, sortBy, sortOrder]);

  // 监听新邮件事件，自动刷新邮件列表
  useEffect(() => {
    if (newEmailCount > 0 && !externalEmails) {
      // 有新邮件时，重新加载第一页
      loadEmails(1, false);
      // 清除新邮件计数
      clearNewEmailCount();
    }
  }, [newEmailCount, externalEmails, loadEmails, clearNewEmailCount]);

  // 获取当前显示的标题
  const getTitle = () => {
    if (title) return title;
    if (selectedFolder) {
      return selectedFolder.display_name;
    }
    if (selectedAccount) {
      return selectedAccount.name;
    }
    return '全部收件';
  };

  // 获取当前使用的邮件数据
  const currentEmails = externalEmails || emails;
  const currentIsLoading = externalLoading !== undefined ? externalLoading : isLoading;
  const currentTotal = totalCount !== undefined ? totalCount : total;

  // 处理邮件点击
  const handleEmailClick = (email: Email) => {
    console.log('📧 [EmailList] 邮件被点击:', {
      emailId: email.id,
      subject: email.subject,
      hasOnEmailSelect: !!onEmailSelect,
    });

    if (onEmailSelect) {
      console.log('📧 [EmailList] 使用外部 onEmailSelect 回调');
      onEmailSelect(email.id);
    } else {
      console.log('📧 [EmailList] 没有提供 onEmailSelect 回调');
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* 邮件列表头部 */}
      <EmailListHeader title={getTitle()} totalCount={currentTotal} />

      {/* 邮件列表内容 */}
      <div className="flex-1 overflow-y-auto">
        {currentIsLoading ? (
          <LoadingSkeleton count={10} />
        ) : currentEmails.length > 0 ? (
          <div>
            {currentEmails.map((email, index) => (
              <EmailItem
                key={email.id}
                email={email}
                isSelected={selectedEmailId === email.id}
                onClick={() => handleEmailClick(email)}
                ref={index === currentEmails.length - 1 ? lastElementRef : undefined}
              />
            ))}

            {/* 加载更多指示器 */}
            {isLoadingMore && (
              <div className="p-4 text-center">
                <div className="text-sm text-gray-500 dark:text-gray-400">加载更多邮件...</div>
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-8 h-8 text-gray-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                暂无邮件
              </h3>
              <p className="text-gray-500 dark:text-gray-400">
                {searchQuery ? '没有找到匹配的邮件' : '此文件夹中暂无邮件'}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* 分页控件 */}
      {showPagination && totalPages > 1 && (
        <div className="border-t border-gray-200 dark:border-gray-700 p-4">
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-500 dark:text-gray-400">
              第 {currentPage} 页，共 {totalPages} 页
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => onPageChange?.(currentPage - 1)}
                disabled={currentPage <= 1}
              >
                <ChevronLeft className="w-4 h-4" />
                上一页
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={() => onPageChange?.(currentPage + 1)}
                disabled={currentPage >= totalPages}
              >
                下一页
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* 邮件右键菜单 */}
      <EmailContextMenu />
    </div>
  );
}
