'use client';

import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import {
  ChevronDown,
  MoreHorizontal,
  Archive,
  Trash2,
  CheckCheck,
  Star,
  FolderOpen,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { useMailboxStore } from '@/lib/store';
import { BulkActions } from './bulk-actions';

interface EmailListHeaderProps {
  title?: string;
  totalCount?: number;
}

export function EmailListHeader({ title = '收件箱', totalCount = 0 }: EmailListHeaderProps) {
  const [showSortMenu, setShowSortMenu] = useState(false);
  const checkboxRef = useRef<HTMLButtonElement>(null);

  const { emails, selectedEmails, sortBy, sortOrder, selectAllEmails, clearSelection, setSort } =
    useMailboxStore();

  // 使用useMemo优化计算，避免不必要的重计算
  const selectionState = useMemo(() => {
    const hasSelection = selectedEmails.size > 0;
    const isPartiallySelected = selectedEmails.size > 0 && selectedEmails.size < emails.length;
    const allSelected = emails.length > 0 && selectedEmails.size === emails.length;

    return {
      hasSelection,
      isPartiallySelected,
      allSelected,
    };
  }, [selectedEmails.size, emails.length]);

  // 使用useCallback避免函数重新创建导致的无限循环
  const handleSelectAll = useCallback(() => {
    // 添加防护条件，避免在emails为空时执行操作
    if (emails.length === 0) return;

    if (selectionState.allSelected) {
      clearSelection();
    } else {
      selectAllEmails();
    }
  }, [emails.length, selectionState.allSelected, clearSelection, selectAllEmails]);

  // 处理排序
  const handleSort = (newSortBy: string) => {
    const newSortOrder = sortBy === newSortBy && sortOrder === 'desc' ? 'asc' : 'desc';
    setSort(newSortBy, newSortOrder);
    setShowSortMenu(false);
  };

  // 获取排序显示文本
  const getSortText = () => {
    const sortLabels: Record<string, string> = {
      date: '时间',
      subject: '主题',
      from: '发件人',
      size: '大小',
    };
    const orderText = sortOrder === 'desc' ? '降序' : '升序';
    return `${sortLabels[sortBy] || '时间'} ${orderText}`;
  };

  // 使用useEffect来设置indeterminate状态，避免在render中直接操作DOM
  useEffect(() => {
    if (checkboxRef.current) {
      const checkbox = checkboxRef.current.querySelector(
        'input[type="checkbox"]'
      ) as HTMLInputElement;
      if (checkbox) {
        checkbox.indeterminate = selectionState.isPartiallySelected;
      }
    }
  }, [selectionState.isPartiallySelected]);

  return (
    <div className="flex-shrink-0 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
      {/* 主要头部区域 */}
      <div className="p-4">
        <div className="flex items-center justify-between mb-3">
          {/* 左侧：标题和统计 */}
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">{title}</h2>
            <span className="text-sm text-gray-500 dark:text-gray-400">{totalCount} 封邮件</span>
          </div>

          {/* 右侧：排序和更多操作 */}
          <div className="flex items-center gap-2">
            {/* 排序按钮 */}
            <div className="relative">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowSortMenu(!showSortMenu)}
                className="gap-1"
              >
                <span className="text-xs">{getSortText()}</span>
                <ChevronDown className="w-3 h-3" />
              </Button>

              {/* 排序菜单 */}
              {showSortMenu && (
                <div className="absolute right-0 top-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg z-10 min-w-[120px]">
                  <div className="py-1">
                    {[
                      { key: 'date', label: '时间' },
                      { key: 'subject', label: '主题' },
                      { key: 'from', label: '发件人' },
                      { key: 'size', label: '大小' },
                    ].map((option) => (
                      <button
                        key={option.key}
                        onClick={() => handleSort(option.key)}
                        className={`
                          w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700
                          ${sortBy === option.key ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300' : 'text-gray-700 dark:text-gray-300'}
                        `}
                      >
                        {option.label}
                        {sortBy === option.key && (
                          <span className="ml-2 text-xs">{sortOrder === 'desc' ? '↓' : '↑'}</span>
                        )}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* 选择和批量操作区域 */}
        <div className="flex items-center justify-between">
          {/* 左侧：全选复选框 */}
          <div className="flex items-center gap-3">
            <Checkbox
              ref={checkboxRef}
              checked={selectionState.allSelected}
              onCheckedChange={handleSelectAll}
              className="data-[state=checked]:bg-blue-600 data-[state=checked]:border-blue-600"
            />
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {selectionState.hasSelection ? `已选择 ${selectedEmails.size} 封邮件` : '全选'}
            </span>
          </div>

          {/* 右侧：批量操作按钮 */}
          {selectionState.hasSelection && <BulkActions selectedCount={selectedEmails.size} />}
        </div>
      </div>

      {/* 点击外部关闭排序菜单 */}
      {showSortMenu && <div className="fixed inset-0 z-0" onClick={() => setShowSortMenu(false)} />}
    </div>
  );
}
