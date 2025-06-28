/**
 * 代码质量检查工具
 * 运行时代码质量监控和建议
 */

// 组件性能分析
export class ComponentPerformanceAnalyzer {
  private static renderTimes: Map<string, number[]> = new Map();
  private static mountTimes: Map<string, number> = new Map();

  static trackRender(componentName: string, renderTime: number) {
    if (process.env.NODE_ENV !== 'development') return;

    if (!this.renderTimes.has(componentName)) {
      this.renderTimes.set(componentName, []);
    }

    this.renderTimes.get(componentName)!.push(renderTime);

    // 如果渲染时间超过16ms（60fps），发出警告
    if (renderTime > 16) {
      console.warn(`🐌 Slow render detected: ${componentName} took ${renderTime.toFixed(2)}ms`);
    }
  }

  static trackMount(componentName: string, mountTime: number) {
    if (process.env.NODE_ENV !== 'development') return;

    this.mountTimes.set(componentName, mountTime);

    // 如果挂载时间超过100ms，发出警告
    if (mountTime > 100) {
      console.warn(`🐌 Slow mount detected: ${componentName} took ${mountTime.toFixed(2)}ms`);
    }
  }

  static getReport() {
    if (process.env.NODE_ENV !== 'development') return null;

    const report: Record<string, any> = {};

    // 渲染性能报告
    for (const [component, times] of this.renderTimes.entries()) {
      const avg = times.reduce((a, b) => a + b, 0) / times.length;
      const max = Math.max(...times);
      const slowRenders = times.filter((t) => t > 16).length;

      report[component] = {
        avgRenderTime: avg.toFixed(2),
        maxRenderTime: max.toFixed(2),
        totalRenders: times.length,
        slowRenders,
        mountTime: this.mountTimes.get(component)?.toFixed(2) || 'N/A',
      };
    }

    return report;
  }

  static getSuggestions() {
    if (process.env.NODE_ENV !== 'development') return [];

    const suggestions: string[] = [];

    for (const [component, times] of this.renderTimes.entries()) {
      const avg = times.reduce((a, b) => a + b, 0) / times.length;
      const slowRenders = times.filter((t) => t > 16).length;
      const renderCount = times.length;

      if (avg > 10) {
        suggestions.push(`${component}: Consider memoization (avg render: ${avg.toFixed(2)}ms)`);
      }

      if (slowRenders > renderCount * 0.3) {
        suggestions.push(
          `${component}: High percentage of slow renders (${slowRenders}/${renderCount})`
        );
      }

      if (renderCount > 50) {
        suggestions.push(
          `${component}: High render count (${renderCount}), check for unnecessary re-renders`
        );
      }
    }

    return suggestions;
  }
}

// 内存泄漏检测
export class MemoryLeakDetector {
  private static listeners: Map<string, number> = new Map();
  private static timers: Map<string, number> = new Map();
  private static intervals: Map<string, number> = new Map();

  static trackEventListener(component: string, event: string, add: boolean) {
    if (process.env.NODE_ENV !== 'development') return;

    const key = `${component}:${event}`;
    const current = this.listeners.get(key) || 0;

    if (add) {
      this.listeners.set(key, current + 1);
    } else {
      this.listeners.set(key, Math.max(0, current - 1));
    }

    // 检查是否有未清理的监听器
    const count = this.listeners.get(key) || 0;
    if (count > 10) {
      console.warn(`🔥 Potential memory leak: ${key} has ${count} listeners`);
    }
  }

  static trackTimer(component: string, type: 'timeout' | 'interval', add: boolean) {
    if (process.env.NODE_ENV !== 'development') return;

    const map = type === 'timeout' ? this.timers : this.intervals;
    const current = map.get(component) || 0;

    if (add) {
      map.set(component, current + 1);
    } else {
      map.set(component, Math.max(0, current - 1));
    }

    // 检查是否有未清理的定时器
    const count = map.get(component) || 0;
    if (count > 5) {
      console.warn(`🔥 Potential memory leak: ${component} has ${count} ${type}s`);
    }
  }

  static getReport() {
    if (process.env.NODE_ENV !== 'development') return null;

    return {
      listeners: Object.fromEntries(this.listeners),
      timers: Object.fromEntries(this.timers),
      intervals: Object.fromEntries(this.intervals),
    };
  }
}

// 依赖数组检查
export function checkDependencyArray(hookName: string, deps: any[], prevDeps?: any[]) {
  if (process.env.NODE_ENV !== 'development') return;

  if (!prevDeps) return;

  // 检查依赖数组长度变化
  if (deps.length !== prevDeps.length) {
    console.warn(
      `⚠️ ${hookName}: Dependency array length changed from ${prevDeps.length} to ${deps.length}`
    );
    return;
  }

  // 检查依赖项变化
  const changes: Array<{ index: number; from: any; to: any }> = [];

  for (let i = 0; i < deps.length; i++) {
    if (deps[i] !== prevDeps[i]) {
      changes.push({
        index: i,
        from: prevDeps[i],
        to: deps[i],
      });
    }
  }

  if (changes.length > 0) {
    console.log(`🔄 ${hookName}: Dependencies changed:`, changes);
  }

  // 检查可能的问题
  deps.forEach((dep, index) => {
    // 检查对象引用
    if (typeof dep === 'object' && dep !== null && !Array.isArray(dep)) {
      console.warn(
        `⚠️ ${hookName}: Object reference in dependency array at index ${index}. Consider using useMemo or useCallback.`
      );
    }

    // 检查函数引用
    if (typeof dep === 'function') {
      console.warn(
        `⚠️ ${hookName}: Function reference in dependency array at index ${index}. Consider using useCallback.`
      );
    }
  });
}

// Props 验证
export function validateProps<T extends Record<string, any>>(
  componentName: string,
  props: T,
  schema: Record<
    keyof T,
    { required?: boolean; type?: string; validator?: (value: any) => boolean }
  >
) {
  if (process.env.NODE_ENV !== 'development') return;

  const errors: string[] = [];

  for (const [key, rules] of Object.entries(schema)) {
    const value = props[key];

    // 检查必需属性
    if (rules.required && (value === undefined || value === null)) {
      errors.push(`Missing required prop: ${key}`);
      continue;
    }

    // 跳过可选的未定义属性
    if (value === undefined || value === null) continue;

    // 检查类型
    if (rules.type && typeof value !== rules.type) {
      errors.push(`Invalid type for prop ${key}: expected ${rules.type}, got ${typeof value}`);
    }

    // 自定义验证
    if (rules.validator && !rules.validator(value)) {
      errors.push(`Custom validation failed for prop: ${key}`);
    }
  }

  if (errors.length > 0) {
    console.error(`❌ ${componentName} prop validation failed:`, errors);
  }
}

// 状态更新检查
export function checkStateUpdates<T>(
  componentName: string,
  newState: T,
  prevState: T,
  stateKey?: string
) {
  if (process.env.NODE_ENV !== 'development') return;

  const key = stateKey || 'state';

  // 检查是否是相同的引用
  if (newState === prevState) {
    console.warn(
      `⚠️ ${componentName}: ${key} update with same reference. This won't trigger re-render.`
    );
    return;
  }

  // 检查深度相等
  if (typeof newState === 'object' && typeof prevState === 'object') {
    if (JSON.stringify(newState) === JSON.stringify(prevState)) {
      console.warn(
        `⚠️ ${componentName}: ${key} update with same content but different reference. Consider using a state updater function.`
      );
    }
  }

  // 检查频繁更新
  const updateKey = `${componentName}:${key}`;
  const now = Date.now();
  const lastUpdate = (checkStateUpdates as any).lastUpdates?.[updateKey] || 0;

  if (!(checkStateUpdates as any).lastUpdates) {
    (checkStateUpdates as any).lastUpdates = {};
  }

  (checkStateUpdates as any).lastUpdates[updateKey] = now;

  if (now - lastUpdate < 16) {
    // 小于一帧的时间
    console.warn(
      `⚠️ ${componentName}: Frequent ${key} updates detected. Consider batching updates.`
    );
  }
}

// 渲染优化建议
export function analyzeRenderOptimization(componentName: string, props: any, prevProps?: any) {
  if (process.env.NODE_ENV !== 'development') return;

  if (!prevProps) return;

  const suggestions: string[] = [];

  // 检查props变化
  const changedProps = Object.keys(props).filter((key) => props[key] !== prevProps[key]);

  if (changedProps.length === 0) {
    suggestions.push('Component re-rendered with same props. Consider React.memo()');
  }

  // 检查函数props
  const functionProps = Object.keys(props).filter((key) => typeof props[key] === 'function');
  if (functionProps.length > 0) {
    suggestions.push(
      `Function props detected: ${functionProps.join(', ')}. Consider useCallback in parent component`
    );
  }

  // 检查对象props
  const objectProps = Object.keys(props).filter(
    (key) => typeof props[key] === 'object' && props[key] !== null && !Array.isArray(props[key])
  );
  if (objectProps.length > 0) {
    suggestions.push(
      `Object props detected: ${objectProps.join(', ')}. Consider useMemo in parent component`
    );
  }

  if (suggestions.length > 0) {
    console.log(`💡 ${componentName} optimization suggestions:`, suggestions);
  }
}

// 代码质量报告
export function generateQualityReport() {
  if (process.env.NODE_ENV !== 'development') return null;

  const report = {
    performance: ComponentPerformanceAnalyzer.getReport(),
    suggestions: ComponentPerformanceAnalyzer.getSuggestions(),
    memoryLeaks: MemoryLeakDetector.getReport(),
    timestamp: new Date().toISOString(),
  };

  console.group('📊 Code Quality Report');
  console.log('Performance:', report.performance);
  console.log('Suggestions:', report.suggestions);
  console.log('Memory Leaks:', report.memoryLeaks);
  console.groupEnd();

  return report;
}

// 自动化质量检查
export function startQualityMonitoring() {
  if (process.env.NODE_ENV !== 'development') return;

  // 每30秒生成一次报告
  const interval = setInterval(() => {
    const suggestions = ComponentPerformanceAnalyzer.getSuggestions();
    if (suggestions.length > 0) {
      console.group('🔍 Quality Check');
      suggestions.forEach((suggestion) => console.log(`💡 ${suggestion}`));
      console.groupEnd();
    }
  }, 30000);

  // 返回清理函数
  return () => clearInterval(interval);
}

// 导出开发工具到全局
if (process.env.NODE_ENV === 'development' && typeof window !== 'undefined') {
  (window as any).__QUALITY_TOOLS__ = {
    performance: ComponentPerformanceAnalyzer,
    memoryLeaks: MemoryLeakDetector,
    generateReport: generateQualityReport,
    startMonitoring: startQualityMonitoring,
  };
}
