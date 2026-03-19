'use client';

import {
  useEffect,
  useCallback,
  useState,
  createContext,
  useContext,
  type ReactNode,
} from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { Filter } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { SearchBar } from './search-bar';
import { SearchFilters } from './search-filters';
import { EmailList } from './email-list';
import { EmailDetail } from './email-detail';
import { LoadingSkeleton } from './loading-skeleton';
import { useSearchEmails, type SearchParams } from '@/hooks/use-search-emails';
import { useIsMobile, useResponsive } from '@/hooks/use-responsive';

function useSearchResultsState() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const isMobile = useIsMobile();
  const [showMobileFilters, setShowMobileFilters] = useState(false);
  const {
    // 搜索状态
    isLoading,
    error,

    // 搜索结果
    emails,
    total,
    page,
    totalPages,

    // 选中状态
    selectedEmailId,
    selectedEmail,

    // 搜索方法
    search,
    updateFilters,
    changePage,
    selectSearchEmail,
    clearSelectedSearchEmail,
  } = useSearchEmails();

  // 调试信息：监控搜索状态变化（仅开发环境）
  useEffect(() => {
    if (process.env.NODE_ENV === 'development') {
      console.log('📄 [SearchResultsPage] 搜索状态更新:', {
        isLoading,
        error: error?.message,
        emailCount: emails.length,
        total,
        page,
        totalPages,
        selectedEmailId,
        hasSelectedEmail: !!selectedEmail,
        timestamp: new Date().toISOString(),
      });
    }
  }, [isLoading, error, emails.length, total, page, totalPages, selectedEmailId, selectedEmail]);

  // 从URL参数初始化搜索
  useEffect(() => {
    const urlQuery = searchParams.get('q');
    console.log('📄 [SearchResultsPage] URL参数变化:', { urlQuery });
    if (urlQuery?.trim()) {
      console.log('📄 [SearchResultsPage] 从URL初始化搜索:', urlQuery);
      search(urlQuery);
    }
  }, [searchParams, search]);

  // 处理筛选条件变化
  const handleFiltersChange = useCallback(
    (newFilters: Partial<SearchParams>) => {
      console.log('📄 [SearchResultsPage] 筛选条件变化:', newFilters);
      updateFilters(newFilters);
    },
    [updateFilters]
  );

  // 处理搜索
  const handleSearch = useCallback(
    (searchQuery: string) => {
      console.log('📄 [SearchResultsPage] 执行搜索:', searchQuery);
      search(searchQuery);
    },
    [search]
  );

  // 处理邮件选择
  const handleEmailSelect = useCallback(
    (emailId: number) => {
      console.log('📄 [SearchResultsPage] 选择邮件:', emailId);
      selectSearchEmail(emailId);

      // 如果是移动端，跳转到邮件详情页面
      if (isMobile) {
        router.push(`/mailbox/mobile/email/${emailId}`);
      }
    },
    [selectSearchEmail, isMobile, router]
  );

  // 处理分页
  const handlePageChange = useCallback(
    (newPage: number) => {
      console.log('📄 [SearchResultsPage] 分页变化:', newPage);
      changePage(newPage);
    },
    [changePage]
  );

  // 获取当前搜索查询
  const currentQuery = searchParams.get('q') || '';

  return {
    isLoading,
    error,
    emails,
    total,
    page,
    totalPages,
    selectedEmailId,
    selectedEmail,
    currentQuery,
    showMobileFilters,
    setShowMobileFilters,
    handleFiltersChange,
    handleSearch,
    handleEmailSelect,
    handlePageChange,
    clearSelectedSearchEmail,
  };
}

type SearchResultsContextValue = ReturnType<typeof useSearchResultsState>;

const SearchResultsContext = createContext<SearchResultsContextValue | null>(null);

function useSearchResultsContext() {
  const context = useContext(SearchResultsContext);
  if (!context) {
    throw new Error('SearchResultsContext 未初始化');
  }
  return context;
}

export function SearchResultsProvider({ children }: { children: ReactNode }) {
  const value = useSearchResultsState();
  return <SearchResultsContext.Provider value={value}>{children}</SearchResultsContext.Provider>;
}

export function SearchResultsHeader() {
  const { currentQuery, isLoading, total, handleSearch, setShowMobileFilters } =
    useSearchResultsContext();

  return (
    <div className="flex-shrink-0 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-4">
      <div className="lg:max-w-2xl mx-auto">
        <SearchBar
          onSearch={handleSearch}
          placeholder="搜索邮件..."
          defaultValue={currentQuery}
        />
      </div>

      {/* 搜索结果统计和移动端筛选按钮 */}
      {currentQuery && (
        <div className="lg:max-w-2xl mx-auto mt-2 flex items-center justify-between">
          <div className="text-sm text-gray-600 dark:text-gray-400">
            {isLoading ? '搜索中...' : `找到 ${total} 个结果`}
          </div>

          {/* 移动端筛选按钮 */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowMobileFilters(true)}
            className="lg:hidden"
          >
            <Filter className="w-4 h-4 mr-2" />
            筛选
          </Button>
        </div>
      )}
    </div>
  );
}

export function SearchResultsContent() {
  const {
    isLoading,
    emails,
    selectedEmailId,
    selectedEmail,
    page,
    totalPages,
    total,
    currentQuery,
    clearSelectedSearchEmail,
    handleFiltersChange,
    handleEmailSelect,
    handlePageChange,
  } = useSearchResultsContext();
  const { isTablet, isDesktop } = useResponsive();

  return (
    <>
      {/* 左侧筛选条件 - 1/5宽度，移动端隐藏 */}
      {isDesktop && (
        <div className="hidden lg:block w-1/5 min-w-[250px] flex-shrink-0 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
          <SearchFilters onFiltersChange={handleFiltersChange} />
        </div>
      )}

      {/* 中间邮件列表 - 1/5宽度，移动端全宽 */}
      <div
        className={`border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 ${
          isDesktop ? 'w-full lg:w-1/5 min-w-[250px] lg:max-w-[350px] flex-shrink-0' : 'flex-1 min-w-0 w-full'
        }`}
      >
        {isLoading ? (
          <div className="p-4">
            <LoadingSkeleton />
          </div>
        ) : emails.length > 0 ? (
          <EmailList
            emails={emails}
            selectedEmailId={selectedEmailId}
            onEmailSelect={handleEmailSelect}
            showPagination={true}
            currentPage={page}
            totalPages={totalPages}
            onPageChange={handlePageChange}
            title="搜索结果"
            totalCount={total}
          />
        ) : currentQuery ? (
          <div className="p-8 text-center">
            <div className="text-gray-500 dark:text-gray-400">
              <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                未找到匹配的邮件
              </h3>
              <p className="text-sm">尝试调整搜索关键词或筛选条件</p>
            </div>
          </div>
        ) : (
          <div className="p-8 text-center">
            <div className="text-gray-500 dark:text-gray-400">
              <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                开始搜索
              </h3>
              <p className="text-sm">在上方输入关键词搜索邮件</p>
            </div>
          </div>
        )}
      </div>

      {/* 右侧邮件详情 - 3/5宽度，移动端隐藏 */}
      {isDesktop && (
        <div className="hidden lg:block flex-1 bg-white dark:bg-gray-800">
          {selectedEmail ? (
            <EmailDetail email={selectedEmail} />
          ) : (
            <div className="h-full flex items-center justify-center">
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
                  选择邮件查看详情
                </h3>
                <p className="text-gray-500 dark:text-gray-400">
                  从左侧列表中选择一封邮件来查看详细内容
                </p>
              </div>
            </div>
          )}
        </div>
      )}

      {isTablet && (
        <Sheet open={!!selectedEmail} onOpenChange={(open) => !open && clearSelectedSearchEmail()}>
          <SheetContent side="right" className="w-full sm:max-w-2xl p-0">
            <SheetHeader className="sr-only">
              <SheetTitle>搜索结果邮件详情</SheetTitle>
            </SheetHeader>
            <div className="h-full bg-white dark:bg-gray-800">
              {selectedEmail && <EmailDetail email={selectedEmail} />}
            </div>
          </SheetContent>
        </Sheet>
      )}
    </>
  );
}

export function SearchResultsMobileFilters() {
  const { showMobileFilters, setShowMobileFilters, handleFiltersChange } = useSearchResultsContext();

  if (!showMobileFilters) return null;

  return (
    <div className="fixed inset-0 z-50 lg:hidden">
      {/* 遮罩层 */}
      <div
        className="absolute inset-0 bg-black bg-opacity-50"
        onClick={() => setShowMobileFilters(false)}
      />

      {/* 筛选面板 */}
      <div className="absolute right-0 top-0 h-full w-80 bg-white dark:bg-gray-800 shadow-xl">
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">筛选条件</h3>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowMobileFilters(false)}
            className="p-2"
          >
            ✕
          </Button>
        </div>

        <div className="overflow-y-auto h-full pb-20">
          <SearchFilters onFiltersChange={handleFiltersChange} />
        </div>
      </div>
    </div>
  );
}

export function SearchResultsPage() {
  return (
    <SearchResultsProvider>
      <div className="h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
        <SearchResultsHeader />

        {/* 主要内容区域 */}
        <div className="flex-1 flex overflow-hidden">
          <SearchResultsContent />
        </div>

        <SearchResultsMobileFilters />
      </div>
    </SearchResultsProvider>
  );
}
