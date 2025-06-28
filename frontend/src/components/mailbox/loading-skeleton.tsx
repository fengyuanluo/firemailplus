'use client';

interface LoadingSkeletonProps {
  count?: number;
}

export function LoadingSkeleton({ count = 5 }: LoadingSkeletonProps) {
  return (
    <div className="space-y-0">
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="p-4 border-b border-gray-100 dark:border-gray-700 animate-pulse"
        >
          <div className="flex items-start gap-3 ml-4">
            {/* 复选框占位 */}
            <div className="flex-shrink-0 w-4 h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>

            {/* 邮件内容占位 */}
            <div className="flex-1 min-w-0 space-y-2">
              {/* 第一行：发件人和时间 */}
              <div className="flex items-center justify-between gap-2">
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-32"></div>
                <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-16"></div>
              </div>

              {/* 第二行：主题 */}
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>

              {/* 第三行：预览 */}
              <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-full"></div>
            </div>

            {/* 右侧操作区域占位 */}
            <div className="flex-shrink-0 w-6 h-6 bg-gray-200 dark:bg-gray-700 rounded"></div>
          </div>
        </div>
      ))}
    </div>
  );
}
