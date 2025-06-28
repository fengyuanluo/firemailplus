/**
 * 开发工具和调试辅助
 * 仅在开发环境中使用
 */

// 性能监控
export class PerformanceMonitor {
  private static instance: PerformanceMonitor;
  private metrics: Map<string, number[]> = new Map();
  private observers: PerformanceObserver[] = [];

  static getInstance(): PerformanceMonitor {
    if (!PerformanceMonitor.instance) {
      PerformanceMonitor.instance = new PerformanceMonitor();
    }
    return PerformanceMonitor.instance;
  }

  // 开始监控
  startMonitoring() {
    if (process.env.NODE_ENV !== 'development') return;

    // 监控导航性能
    if ('PerformanceObserver' in window) {
      const navObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          console.log('Navigation:', entry);
        }
      });
      navObserver.observe({ entryTypes: ['navigation'] });
      this.observers.push(navObserver);

      // 监控资源加载
      const resourceObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.duration > 1000) {
            console.warn('Slow resource:', entry.name, entry.duration);
          }
        }
      });
      resourceObserver.observe({ entryTypes: ['resource'] });
      this.observers.push(resourceObserver);

      // 监控长任务
      const longTaskObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          console.warn('Long task detected:', entry.duration);
        }
      });
      longTaskObserver.observe({ entryTypes: ['longtask'] });
      this.observers.push(longTaskObserver);
    }
  }

  // 记录性能指标
  recordMetric(name: string, value: number) {
    if (process.env.NODE_ENV !== 'development') return;

    if (!this.metrics.has(name)) {
      this.metrics.set(name, []);
    }
    this.metrics.get(name)!.push(value);
  }

  // 获取性能报告
  getReport() {
    if (process.env.NODE_ENV !== 'development') return null;

    const report: Record<string, any> = {};

    for (const [name, values] of this.metrics.entries()) {
      const avg = values.reduce((a, b) => a + b, 0) / values.length;
      const min = Math.min(...values);
      const max = Math.max(...values);

      report[name] = { avg, min, max, count: values.length };
    }

    return report;
  }

  // 停止监控
  stopMonitoring() {
    this.observers.forEach((observer) => observer.disconnect());
    this.observers = [];
  }
}

// 组件渲染追踪
export function trackRender(componentName: string) {
  if (process.env.NODE_ENV !== 'development') return () => {};

  const startTime = performance.now();
  let renderCount = 0;

  return () => {
    renderCount++;
    const endTime = performance.now();
    const duration = endTime - startTime;

    if (renderCount > 10) {
      console.warn(`${componentName} rendered ${renderCount} times in ${duration.toFixed(2)}ms`);
    }

    PerformanceMonitor.getInstance().recordMetric(`${componentName}_render`, duration);
  };
}

// 内存使用监控
export function monitorMemory() {
  if (process.env.NODE_ENV !== 'development') return;
  if (!('memory' in performance)) return;

  const memory = (performance as any).memory;

  console.log('Memory usage:', {
    used: `${(memory.usedJSHeapSize / 1024 / 1024).toFixed(2)} MB`,
    total: `${(memory.totalJSHeapSize / 1024 / 1024).toFixed(2)} MB`,
    limit: `${(memory.jsHeapSizeLimit / 1024 / 1024).toFixed(2)} MB`,
  });
}

// 网络请求监控
export function monitorNetworkRequests() {
  if (process.env.NODE_ENV !== 'development') return;

  const originalFetch = window.fetch;

  window.fetch = async (...args) => {
    const startTime = performance.now();
    const url = typeof args[0] === 'string' ? args[0] : (args[0] as Request).url;

    try {
      const response = await originalFetch(...args);
      const endTime = performance.now();
      const duration = endTime - startTime;

      console.log(`Fetch: ${url} - ${response.status} - ${duration.toFixed(2)}ms`);

      if (duration > 2000) {
        console.warn(`Slow request: ${url} took ${duration.toFixed(2)}ms`);
      }

      return response;
    } catch (error) {
      const endTime = performance.now();
      const duration = endTime - startTime;

      console.error(`Fetch failed: ${url} - ${duration.toFixed(2)}ms`, error);
      throw error;
    }
  };
}

// 状态变化追踪
export function trackStateChanges<T>(storeName: string, state: T, prevState: T) {
  if (process.env.NODE_ENV !== 'development') return;

  const changes: Record<string, { from: any; to: any }> = {};

  for (const key in state) {
    if (state[key] !== prevState[key]) {
      changes[key] = {
        from: prevState[key],
        to: state[key],
      };
    }
  }

  if (Object.keys(changes).length > 0) {
    console.log(`${storeName} state changed:`, changes);
  }
}

// 错误边界增强
export function enhanceErrorBoundary(error: Error, errorInfo: any) {
  if (process.env.NODE_ENV !== 'development') return;

  console.group('Error Boundary Caught Error');
  console.error('Error:', error);
  console.error('Error Info:', errorInfo);
  console.error('Stack:', error.stack);
  console.groupEnd();

  // 发送错误报告到开发服务器
  if (process.env.NODE_ENV === 'development') {
    fetch('/api/dev/error-report', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        error: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
        timestamp: new Date().toISOString(),
      }),
    }).catch(() => {
      // 忽略报告失败
    });
  }
}

// 开发者控制台命令
export function setupDevConsole() {
  if (process.env.NODE_ENV !== 'development') return;
  if (typeof window === 'undefined') return;

  // 添加全局开发工具
  (window as any).__DEV_TOOLS__ = {
    // 性能监控
    performance: PerformanceMonitor.getInstance(),

    // 内存监控
    memory: monitorMemory,

    // 清理控制台
    clear: () => console.clear(),

    // 获取应用状态
    getState: () => {
      // 这里可以返回所有store的状态
      return {
        // auth: useAuthStore.getState(),
        // mailbox: useMailboxStore.getState(),
        // ui: useUIStore.getState(),
      };
    },

    // 模拟网络延迟
    simulateNetworkDelay: (delay: number) => {
      const originalFetch = window.fetch;
      window.fetch = async (...args) => {
        await new Promise((resolve) => setTimeout(resolve, delay));
        return originalFetch(...args);
      };
      console.log(`Network delay set to ${delay}ms`);
    },

    // 模拟网络错误
    simulateNetworkError: (probability: number = 0.5) => {
      const originalFetch = window.fetch;
      window.fetch = async (...args) => {
        if (Math.random() < probability) {
          throw new Error('Simulated network error');
        }
        return originalFetch(...args);
      };
      console.log(`Network error probability set to ${probability * 100}%`);
    },

    // 重置网络模拟
    resetNetworkSimulation: () => {
      // 重新加载页面来重置fetch
      window.location.reload();
    },
  };

  console.log('🛠️ Dev tools available at window.__DEV_TOOLS__');
  console.log('Available commands:');
  console.log('- __DEV_TOOLS__.performance.getReport() - Get performance report');
  console.log('- __DEV_TOOLS__.memory() - Check memory usage');
  console.log('- __DEV_TOOLS__.getState() - Get application state');
  console.log('- __DEV_TOOLS__.simulateNetworkDelay(1000) - Add network delay');
  console.log('- __DEV_TOOLS__.simulateNetworkError(0.3) - Simulate network errors');
}

// 组件树可视化
export function visualizeComponentTree() {
  if (process.env.NODE_ENV !== 'development') return;

  // 这个功能需要React DevTools的支持
  console.log('Use React DevTools for component tree visualization');
}

// 热重载增强
export function enhanceHotReload() {
  if (process.env.NODE_ENV !== 'development') return;
  if (typeof module === 'undefined' || !(module as any).hot) return;

  // 保存状态到sessionStorage
  const saveState = () => {
    try {
      const state = {
        // auth: useAuthStore.getState(),
        // mailbox: useMailboxStore.getState(),
        timestamp: Date.now(),
      };
      sessionStorage.setItem('__DEV_STATE__', JSON.stringify(state));
    } catch (error) {
      console.warn('Failed to save state for hot reload:', error);
    }
  };

  // 恢复状态
  const restoreState = () => {
    try {
      const savedState = sessionStorage.getItem('__DEV_STATE__');
      if (savedState) {
        const state = JSON.parse(savedState);
        // 这里可以恢复各个store的状态
        console.log('State restored from hot reload:', state);
      }
    } catch (error) {
      console.warn('Failed to restore state from hot reload:', error);
    }
  };

  // 监听热重载事件
  (module as any).hot.accept(() => {
    console.log('Hot reload triggered');
    saveState();
  });

  // 页面加载时恢复状态
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', restoreState);
  } else {
    restoreState();
  }
}

// 初始化开发工具
export function initDevTools() {
  if (process.env.NODE_ENV !== 'development') return;

  console.log('🚀 Initializing development tools...');

  // 启动性能监控
  PerformanceMonitor.getInstance().startMonitoring();

  // 监控网络请求
  monitorNetworkRequests();

  // 设置开发者控制台
  setupDevConsole();

  // 增强热重载
  enhanceHotReload();

  // 定期内存检查
  setInterval(monitorMemory, 30000);

  console.log('✅ Development tools initialized');
}

// 清理开发工具
export function cleanupDevTools() {
  if (process.env.NODE_ENV !== 'development') return;

  PerformanceMonitor.getInstance().stopMonitoring();

  // 清理全局变量
  if (typeof window !== 'undefined') {
    delete (window as any).__DEV_TOOLS__;
  }
}
