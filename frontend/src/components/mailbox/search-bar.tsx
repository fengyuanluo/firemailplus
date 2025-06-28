'use client';

import { useState, useRef, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Search, X, Filter, Menu, Plus } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useMailboxStore, useUIStore } from '@/lib/store';

interface SearchBarProps {
  onSearch?: (query: string) => void;
  placeholder?: string;
  defaultValue?: string;
  showAdvancedSearch?: boolean;
}

export function SearchBar({
  onSearch,
  placeholder = '搜索邮件、发件人、主题...',
  defaultValue = '',
  showAdvancedSearch = true,
}: SearchBarProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [localQuery, setLocalQuery] = useState(defaultValue);
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  const { searchQuery, setSearchQuery } = useMailboxStore();
  const { toggleSidebar, isMobile } = useUIStore();

  // 同步搜索查询和默认值
  useEffect(() => {
    if (defaultValue) {
      setLocalQuery(defaultValue);
    } else {
      setLocalQuery(searchQuery);
    }
  }, [searchQuery, defaultValue]);

  const handleSearch = () => {
    if (onSearch) {
      // 如果提供了外部搜索处理函数，使用它
      onSearch(localQuery);
    } else {
      // 否则使用默认行为：更新store并导航到搜索页面
      setSearchQuery(localQuery);
      if (localQuery.trim()) {
        router.push(`/mailbox/search?q=${encodeURIComponent(localQuery)}`);
      }
    }
  };

  const handleClear = () => {
    setLocalQuery('');
    setSearchQuery('');
    setIsExpanded(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    } else if (e.key === 'Escape') {
      handleClear();
    }
  };

  const handleFocus = () => {
    setIsExpanded(true);
  };

  const handleBlur = () => {
    if (!localQuery) {
      setIsExpanded(false);
    }
  };

  const handleAddEmail = () => {
    router.push('/add-email');
  };

  return (
    <div className="px-4 py-3">
      <div className="flex items-center gap-3 max-w-4xl mx-auto">
        {/* 移动端菜单按钮 */}
        {isMobile && (
          <Button variant="ghost" size="sm" onClick={toggleSidebar} className="p-2 h-auto">
            <Menu className="w-5 h-5" />
          </Button>
        )}

        {/* 搜索框容器 */}
        <div className="flex-1 relative">
          <div
            className={`
            relative flex items-center transition-all duration-200
            ${isExpanded ? 'bg-gray-100 dark:bg-gray-700' : 'bg-gray-100 dark:bg-gray-700'}
            rounded-lg border border-gray-200 dark:border-gray-600
            ${isExpanded ? 'ring-2 ring-gray-900 dark:ring-gray-100' : ''}
          `}
          >
            {/* 搜索图标 */}
            <div className="pl-3 pr-2">
              <Search className="w-4 h-4 text-gray-500 dark:text-gray-400" />
            </div>

            {/* 搜索输入框 */}
            <Input
              ref={inputRef}
              type="text"
              placeholder={placeholder}
              value={localQuery}
              onChange={(e) => setLocalQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              onFocus={handleFocus}
              onBlur={handleBlur}
              className="flex-1 border-0 bg-transparent focus:ring-0 focus:outline-none placeholder:text-gray-500 dark:placeholder:text-gray-400"
            />

            {/* 清除按钮 */}
            {localQuery && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClear}
                className="p-1 h-auto mr-2 hover:bg-gray-200 dark:hover:bg-gray-600"
              >
                <X className="w-4 h-4" />
              </Button>
            )}
          </div>

          {/* 搜索建议下拉框 */}
          {isExpanded && localQuery && (
            <div className="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg z-50">
              <div className="p-2">
                <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">搜索建议</div>
                <div className="space-y-1">
                  <button
                    onClick={() => {
                      setLocalQuery(`from:${localQuery}`);
                      inputRef.current?.focus();
                    }}
                    className="w-full text-left px-2 py-1 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                  >
                    发件人: {localQuery}
                  </button>
                  <button
                    onClick={() => {
                      setLocalQuery(`subject:${localQuery}`);
                      inputRef.current?.focus();
                    }}
                    className="w-full text-left px-2 py-1 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                  >
                    主题: {localQuery}
                  </button>
                  <button
                    onClick={() => {
                      setLocalQuery(`body:${localQuery}`);
                      inputRef.current?.focus();
                    }}
                    className="w-full text-left px-2 py-1 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                  >
                    正文: {localQuery}
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* 高级搜索按钮 */}
        {showAdvancedSearch && (
          <Button
            variant="ghost"
            size="sm"
            className="p-2 h-auto"
            title="高级搜索"
            onClick={() => router.push('/mailbox/search')}
          >
            <Filter className="w-4 h-4" />
          </Button>
        )}

        {/* 添加邮箱按钮 - 只在桌面端显示 */}
        {!isMobile && (
          <Button
            variant="outline"
            size="sm"
            className="gap-2 h-9 px-3 border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700"
            title="添加邮箱账户"
            onClick={handleAddEmail}
          >
            <Plus className="w-4 h-4" />
            <span className="text-sm">添加邮箱</span>
          </Button>
        )}
      </div>
    </div>
  );
}
