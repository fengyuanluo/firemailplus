/**
 * 水合加载组件
 * 在水合过程中显示的加载界面
 */

interface HydrationLoaderProps {
  message?: string;
  className?: string;
}

export function HydrationLoader({
  message = '正在初始化应用...',
  className = '',
}: HydrationLoaderProps) {
  return (
    <div className={`min-h-screen flex items-center justify-center ${className}`}>
      <div className="text-center">
        <div className="w-8 h-8 border-2 border-gray-900 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
        <p className="text-gray-600">{message}</p>
      </div>
    </div>
  );
}
