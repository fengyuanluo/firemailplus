'use client';

import { useEffect, useState } from 'react';
import { Menu, X, Monitor, Smartphone } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useUIStore } from '@/lib/store';
import { useIsMobile } from '@/hooks/use-responsive';
import { ProtectedRoute } from '@/components/auth/route-guard';

export default function TestSidebarPage() {
  const isMobile = useIsMobile();
  const { sidebarOpen, sidebarOpenMobile, toggleSidebar, setSidebarOpen, setSidebarOpenMobile } =
    useUIStore();

  const [windowWidth, setWindowWidth] = useState(0);

  useEffect(() => {
    const updateWidth = () => setWindowWidth(window.innerWidth);
    updateWidth();
    window.addEventListener('resize', updateWidth);
    return () => window.removeEventListener('resize', updateWidth);
  }, []);

  const currentSidebarOpen = isMobile ? sidebarOpenMobile : sidebarOpen;

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        {/* 顶部状态栏 */}
        <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-4">
          <div className="max-w-4xl mx-auto">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
              侧边栏状态测试
            </h1>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 text-sm">
              <div className="bg-gray-50 dark:bg-gray-700 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <Monitor className="w-4 h-4" />
                  <span className="font-medium">设备类型</span>
                </div>
                <div className="text-lg font-mono">{isMobile ? '📱 移动端' : '🖥️ 桌面端'}</div>
                <div className="text-xs text-gray-500 mt-1">宽度: {windowWidth}px</div>
              </div>

              <div className="bg-gray-50 dark:bg-gray-700 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <span className="font-medium">桌面端侧边栏</span>
                </div>
                <div className="text-lg font-mono">{sidebarOpen ? '✅ 打开' : '❌ 关闭'}</div>
              </div>

              <div className="bg-gray-50 dark:bg-gray-700 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <Smartphone className="w-4 h-4" />
                  <span className="font-medium">移动端侧边栏</span>
                </div>
                <div className="text-lg font-mono">{sidebarOpenMobile ? '✅ 打开' : '❌ 关闭'}</div>
              </div>

              <div className="bg-gray-50 dark:bg-gray-700 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <span className="font-medium">当前状态</span>
                </div>
                <div className="text-lg font-mono">
                  {currentSidebarOpen ? '✅ 显示' : '❌ 隐藏'}
                </div>
              </div>
            </div>

            <div className="flex gap-3 mt-4">
              <Button onClick={toggleSidebar} className="flex items-center gap-2">
                {currentSidebarOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
                切换侧边栏
              </Button>

              <Button
                variant="outline"
                onClick={() => setSidebarOpen(!sidebarOpen)}
                disabled={isMobile}
              >
                切换桌面端
              </Button>

              <Button
                variant="outline"
                onClick={() => setSidebarOpenMobile(!sidebarOpenMobile)}
                disabled={!isMobile}
              >
                切换移动端
              </Button>
            </div>
          </div>
        </div>

        {/* 主要内容区域 */}
        <div className="flex overflow-hidden" style={{ height: 'calc(100vh - 140px)' }}>
          {/* 移动端遮罩层 */}
          {isMobile && currentSidebarOpen && (
            <div
              className="fixed inset-0 bg-black bg-opacity-50 z-40 md:hidden"
              onClick={toggleSidebar}
            />
          )}

          {/* 侧边栏 */}
          <div
            className={`
          flex-shrink-0 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700
          ${
            isMobile
              ? `fixed left-0 top-0 h-full w-80 z-50 transform transition-transform duration-300 ${
                  currentSidebarOpen ? 'translate-x-0' : '-translate-x-full'
                }`
              : `w-80 ${currentSidebarOpen ? 'block' : 'hidden'}`
          }
        `}
          >
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  侧边栏内容
                </h2>
                {isMobile && (
                  <Button variant="ghost" size="sm" onClick={toggleSidebar}>
                    <X className="w-4 h-4" />
                  </Button>
                )}
              </div>

              <div className="space-y-4">
                <div className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-2">测试说明</h3>
                  <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
                    <li>• 桌面端：侧边栏默认显示，可以切换</li>
                    <li>• 移动端：侧边栏默认隐藏，左拉显示</li>
                    <li>• 状态独立存储，互不影响</li>
                    <li>• 点击遮罩层可关闭移动端侧边栏</li>
                  </ul>
                </div>

                <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <h3 className="font-medium text-blue-900 dark:text-blue-100 mb-2">当前设备</h3>
                  <p className="text-sm text-blue-700 dark:text-blue-300">
                    {isMobile ? '移动端模式' : '桌面端模式'}
                  </p>
                  <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                    侧边栏状态: {currentSidebarOpen ? '显示' : '隐藏'}
                  </p>
                </div>

                <div className="space-y-2">
                  <div className="p-3 bg-gray-100 dark:bg-gray-600 rounded text-sm">菜单项 1</div>
                  <div className="p-3 bg-gray-100 dark:bg-gray-600 rounded text-sm">菜单项 2</div>
                  <div className="p-3 bg-gray-100 dark:bg-gray-600 rounded text-sm">菜单项 3</div>
                </div>
              </div>
            </div>
          </div>

          {/* 主要内容 */}
          <div className="flex-1 p-6 overflow-auto">
            <div className="max-w-2xl">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">
                主要内容区域
              </h2>

              <div className="prose dark:prose-invert">
                <p>
                  这是主要内容区域。在移动端，当侧边栏打开时，主要内容会被遮罩层覆盖。
                  在桌面端，侧边栏和主要内容并排显示。
                </p>

                <h3>测试步骤：</h3>
                <ol>
                  <li>在桌面端（宽度 &gt; 768px）测试侧边栏切换</li>
                  <li>调整浏览器窗口到移动端尺寸（宽度 &lt; 768px）</li>
                  <li>观察侧边栏是否自动隐藏</li>
                  <li>点击"切换侧边栏"按钮测试移动端侧边栏</li>
                  <li>点击遮罩层测试关闭功能</li>
                </ol>

                <h3>预期行为：</h3>
                <ul>
                  <li>桌面端：侧边栏默认显示，状态持久化</li>
                  <li>移动端：侧边栏默认隐藏，状态独立存储</li>
                  <li>切换设备类型时，使用对应的侧边栏状态</li>
                  <li>移动端侧边栏有滑动动画和遮罩层</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </ProtectedRoute>
  );
}
