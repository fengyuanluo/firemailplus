/**
 * 性能优化Hook
 * 减少不必要的重渲染，优化组件性能
 */

import { useCallback, useMemo, useRef, useEffect, useState } from 'react';

// 防抖Hook
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

// 节流Hook
export function useThrottle<T>(value: T, limit: number): T {
  const [throttledValue, setThrottledValue] = useState<T>(value);
  const lastRan = useRef<number>(Date.now());

  useEffect(() => {
    const handler = setTimeout(
      () => {
        if (Date.now() - lastRan.current >= limit) {
          setThrottledValue(value);
          lastRan.current = Date.now();
        }
      },
      limit - (Date.now() - lastRan.current)
    );

    return () => {
      clearTimeout(handler);
    };
  }, [value, limit]);

  return throttledValue;
}

// 稳定的回调Hook
export function useStableCallback<T extends (...args: any[]) => any>(callback: T): T {
  const callbackRef = useRef<T>(callback);

  // 更新ref但不触发重渲染
  useEffect(() => {
    callbackRef.current = callback;
  });

  // 返回稳定的回调函数
  return useCallback(
    ((...args: any[]) => {
      return callbackRef.current(...args);
    }) as T,
    []
  );
}

// 深度比较的useMemo
export function useDeepMemo<T>(factory: () => T, deps: any[]): T {
  const ref = useRef<{ deps: any[]; value: T } | null>(null);

  if (!ref.current || !deepEqual(ref.current.deps, deps)) {
    ref.current = {
      deps,
      value: factory(),
    };
  }

  return ref.current.value;
}

// 深度相等比较
function deepEqual(a: any, b: any): boolean {
  if (a === b) return true;

  if (a == null || b == null) return false;

  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) {
      if (!deepEqual(a[i], b[i])) return false;
    }
    return true;
  }

  if (typeof a === 'object' && typeof b === 'object') {
    const keysA = Object.keys(a);
    const keysB = Object.keys(b);

    if (keysA.length !== keysB.length) return false;

    for (const key of keysA) {
      if (!keysB.includes(key)) return false;
      if (!deepEqual(a[key], b[key])) return false;
    }
    return true;
  }

  return false;
}

// 前一个值Hook
export function usePrevious<T>(value: T): T | undefined {
  const ref = useRef<T | undefined>(undefined);

  useEffect(() => {
    ref.current = value;
  });

  return ref.current;
}

// 渲染计数Hook（开发环境调试用）
export function useRenderCount(componentName?: string): number {
  const renderCount = useRef(0);

  renderCount.current++;

  if (process.env.NODE_ENV === 'development' && componentName) {
    console.log(`${componentName} rendered ${renderCount.current} times`);
  }

  return renderCount.current;
}

// 为什么重渲染Hook（开发环境调试用）
export function useWhyDidYouUpdate(name: string, props: Record<string, any>) {
  const previousProps = useRef<Record<string, any> | undefined>(undefined);

  useEffect(() => {
    if (previousProps.current) {
      const allKeys = Object.keys({ ...previousProps.current, ...props });
      const changedProps: Record<string, { from: any; to: any }> = {};

      allKeys.forEach((key) => {
        if (previousProps.current![key] !== props[key]) {
          changedProps[key] = {
            from: previousProps.current![key],
            to: props[key],
          };
        }
      });

      if (Object.keys(changedProps).length) {
        console.log('[why-did-you-update]', name, changedProps);
      }
    }

    previousProps.current = props;
  });
}

// 虚拟化列表Hook
export function useVirtualization<T>(
  items: T[],
  itemHeight: number,
  containerHeight: number,
  overscan = 5
) {
  const [scrollTop, setScrollTop] = useState(0);

  const visibleRange = useMemo(() => {
    const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan);
    const endIndex = Math.min(
      items.length - 1,
      Math.ceil((scrollTop + containerHeight) / itemHeight) + overscan
    );

    return { startIndex, endIndex };
  }, [scrollTop, itemHeight, containerHeight, items.length, overscan]);

  const visibleItems = useMemo(() => {
    return items.slice(visibleRange.startIndex, visibleRange.endIndex + 1).map((item, index) => ({
      item,
      index: visibleRange.startIndex + index,
    }));
  }, [items, visibleRange]);

  const totalHeight = items.length * itemHeight;
  const offsetY = visibleRange.startIndex * itemHeight;

  return {
    visibleItems,
    totalHeight,
    offsetY,
    setScrollTop,
  };
}

// 懒加载Hook
export function useLazyLoad<T>(loadMore: () => Promise<T[]>, hasMore: boolean, threshold = 100) {
  const [isLoading, setIsLoading] = useState(false);
  const [items, setItems] = useState<T[]>([]);
  const observerRef = useRef<IntersectionObserver | null>(null);

  const lastElementRef = useCallback(
    (node: HTMLElement | null) => {
      if (isLoading) return;

      if (observerRef.current) observerRef.current.disconnect();

      observerRef.current = new IntersectionObserver(
        (entries) => {
          if (entries[0].isIntersecting && hasMore) {
            setIsLoading(true);
            loadMore()
              .then((newItems) => {
                setItems((prev) => [...prev, ...newItems]);
                setIsLoading(false);
              })
              .catch(() => {
                setIsLoading(false);
              });
          }
        },
        { rootMargin: `${threshold}px` }
      );

      if (node) observerRef.current.observe(node);
    },
    [isLoading, hasMore, loadMore, threshold]
  );

  return {
    items,
    isLoading,
    lastElementRef,
    setItems,
  };
}

// 批量更新Hook
export function useBatchUpdate<T>(initialValue: T, delay = 100) {
  const [value, setValue] = useState<T>(initialValue);
  const pendingUpdates = useRef<((prev: T) => T)[]>([]);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const batchUpdate = useCallback(
    (updater: (prev: T) => T) => {
      pendingUpdates.current.push(updater);

      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      timeoutRef.current = setTimeout(() => {
        setValue((prev) => {
          let result = prev;
          pendingUpdates.current.forEach((update) => {
            result = update(result);
          });
          pendingUpdates.current = [];
          return result;
        });
      }, delay);
    },
    [delay]
  );

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return [value, batchUpdate] as const;
}

// 内存化选择器Hook
export function useSelector<T, R>(
  source: T,
  selector: (source: T) => R,
  equalityFn?: (a: R, b: R) => boolean
): R {
  const selectorRef = useRef(selector);
  const equalityRef = useRef(equalityFn);
  const resultRef = useRef<R | undefined>(undefined);
  const sourceRef = useRef<T | undefined>(undefined);

  // 更新refs
  selectorRef.current = selector;
  equalityRef.current = equalityFn;

  const newResult = selectorRef.current(source);

  // 如果源数据或结果发生变化，更新结果
  if (
    sourceRef.current !== source ||
    (equalityRef.current
      ? !equalityRef.current(resultRef.current!, newResult)
      : resultRef.current !== newResult)
  ) {
    resultRef.current = newResult;
    sourceRef.current = source;
  }

  return resultRef.current!;
}

// 组件卸载检查Hook
export function useIsMounted() {
  const isMountedRef = useRef(true);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  return useCallback(() => isMountedRef.current, []);
}

// 性能监控Hook
export function usePerformanceMonitor(name: string) {
  const startTime = useRef<number | undefined>(undefined);
  const renderCount = useRef(0);

  // 组件开始渲染
  if (!startTime.current) {
    startTime.current = performance.now();
  }

  renderCount.current++;

  useEffect(() => {
    // 组件渲染完成
    const endTime = performance.now();
    const duration = endTime - startTime.current!;

    if (process.env.NODE_ENV === 'development') {
      console.log(
        `[Performance] ${name}: ${duration.toFixed(2)}ms (${renderCount.current} renders)`
      );
    }

    // 重置计时器
    startTime.current = performance.now();
  });

  return {
    renderCount: renderCount.current,
  };
}
