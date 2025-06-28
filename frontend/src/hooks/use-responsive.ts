/**
 * 统一的响应式设计Hook
 * 提供一致的屏幕尺寸检测和响应式状态管理
 */

import { useState, useEffect, useCallback } from 'react';
import { useUIStore } from '@/lib/store';

// 断点定义
export const BREAKPOINTS = {
  xs: 0,
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  '2xl': 1536,
} as const;

// 屏幕尺寸类型
export type BreakpointKey = keyof typeof BREAKPOINTS;
export type ScreenSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';
export type DeviceSize = 'mobile' | 'tablet' | 'desktop';

// 响应式状态
interface ResponsiveState {
  width: number;
  height: number;
  isMobile: boolean;
  isTablet: boolean;
  isDesktop: boolean;
  currentBreakpoint: ScreenSize;
  orientation: 'portrait' | 'landscape';
}

// 获取当前断点
function getCurrentBreakpoint(width: number): ScreenSize {
  if (width >= BREAKPOINTS['2xl']) return '2xl';
  if (width >= BREAKPOINTS.xl) return 'xl';
  if (width >= BREAKPOINTS.lg) return 'lg';
  if (width >= BREAKPOINTS.md) return 'md';
  if (width >= BREAKPOINTS.sm) return 'sm';
  return 'xs';
}

// 判断设备类型
function getDeviceType(width: number) {
  return {
    isMobile: width < BREAKPOINTS.md, // < 768px
    isTablet: width >= BREAKPOINTS.md && width < BREAKPOINTS.lg, // 768px - 1024px
    isDesktop: width >= BREAKPOINTS.lg, // >= 1024px
  };
}

// 基础响应式Hook
export function useResponsive() {
  const [state, setState] = useState<ResponsiveState>(() => {
    // 服务端渲染时的默认值
    if (typeof window === 'undefined') {
      return {
        width: 1024,
        height: 768,
        isMobile: false,
        isTablet: false,
        isDesktop: true,
        currentBreakpoint: 'lg',
        orientation: 'landscape',
      };
    }

    const width = window.innerWidth;
    const height = window.innerHeight;
    const deviceType = getDeviceType(width);

    return {
      width,
      height,
      ...deviceType,
      currentBreakpoint: getCurrentBreakpoint(width),
      orientation: height > width ? 'portrait' : 'landscape',
    };
  });

  const updateState = useCallback(() => {
    if (typeof window === 'undefined') return;

    const width = window.innerWidth;
    const height = window.innerHeight;
    const deviceType = getDeviceType(width);
    const currentBreakpoint = getCurrentBreakpoint(width);
    const orientation = height > width ? 'portrait' : 'landscape';

    setState((prev) => {
      // 只在状态真正改变时更新
      if (
        prev.width === width &&
        prev.height === height &&
        prev.currentBreakpoint === currentBreakpoint &&
        prev.orientation === orientation
      ) {
        return prev;
      }

      return {
        width,
        height,
        ...deviceType,
        currentBreakpoint,
        orientation,
      };
    });
  }, []);

  useEffect(() => {
    // 初始化时更新一次
    updateState();

    // 防抖处理
    let timeoutId: NodeJS.Timeout;
    const debouncedUpdate = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(updateState, 100);
    };

    window.addEventListener('resize', debouncedUpdate);
    window.addEventListener('orientationchange', debouncedUpdate);

    return () => {
      window.removeEventListener('resize', debouncedUpdate);
      window.removeEventListener('orientationchange', debouncedUpdate);
      clearTimeout(timeoutId);
    };
  }, [updateState]);

  return state;
}

// 移动端检测Hook
export function useIsMobile() {
  const { isMobile } = useResponsive();
  const { setIsMobile } = useUIStore();

  // 同步到全局状态
  useEffect(() => {
    setIsMobile(isMobile);
  }, [isMobile, setIsMobile]);

  return isMobile;
}

// 断点匹配Hook
export function useBreakpoint(breakpoint: BreakpointKey) {
  const { width } = useResponsive();
  return width >= BREAKPOINTS[breakpoint];
}

// 媒体查询Hook
export function useMediaQuery(query: string) {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const mediaQuery = window.matchMedia(query);
    setMatches(mediaQuery.matches);

    const handler = (event: MediaQueryListEvent) => {
      setMatches(event.matches);
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [query]);

  return matches;
}

// 屏幕方向Hook
export function useOrientation() {
  const { orientation } = useResponsive();
  return orientation;
}

// 视口尺寸Hook
export function useViewport() {
  const { width, height } = useResponsive();
  return { width, height };
}

// 响应式值Hook
export function useResponsiveValue<T>(values: Partial<Record<ScreenSize, T>>, defaultValue: T): T {
  const { currentBreakpoint } = useResponsive();

  // 按优先级查找值
  const breakpointOrder: ScreenSize[] = ['2xl', 'xl', 'lg', 'md', 'sm', 'xs'];
  const currentIndex = breakpointOrder.indexOf(currentBreakpoint);

  // 从当前断点开始向下查找
  for (let i = currentIndex; i < breakpointOrder.length; i++) {
    const breakpoint = breakpointOrder[i];
    if (values[breakpoint] !== undefined) {
      return values[breakpoint]!;
    }
  }

  return defaultValue;
}

// 条件渲染Hook
export function useResponsiveRender() {
  const responsive = useResponsive();

  return {
    // 仅在移动端渲染
    mobile: (component: React.ReactNode) => (responsive.isMobile ? component : null),

    // 仅在平板端渲染
    tablet: (component: React.ReactNode) => (responsive.isTablet ? component : null),

    // 仅在桌面端渲染
    desktop: (component: React.ReactNode) => (responsive.isDesktop ? component : null),

    // 根据断点渲染
    breakpoint: (breakpoint: ScreenSize, component: React.ReactNode) =>
      responsive.currentBreakpoint === breakpoint ? component : null,

    // 大于等于指定断点时渲染
    above: (breakpoint: BreakpointKey, component: React.ReactNode) =>
      responsive.width >= BREAKPOINTS[breakpoint] ? component : null,

    // 小于指定断点时渲染
    below: (breakpoint: BreakpointKey, component: React.ReactNode) =>
      responsive.width < BREAKPOINTS[breakpoint] ? component : null,
  };
}

// 响应式类名Hook
export function useResponsiveClassName() {
  const { currentBreakpoint, isMobile, isTablet, isDesktop, orientation } = useResponsive();

  return {
    // 基础类名
    base: `screen-${currentBreakpoint}`,

    // 设备类型类名
    device: isMobile ? 'mobile' : isTablet ? 'tablet' : 'desktop',

    // 方向类名
    orientation,

    // 组合类名
    combined: `screen-${currentBreakpoint} ${isMobile ? 'mobile' : isTablet ? 'tablet' : 'desktop'} ${orientation}`,

    // 条件类名生成器
    when: (condition: boolean, className: string) => (condition ? className : ''),

    // 断点条件类名（需要在组件中使用）
    above: (breakpoint: BreakpointKey, className: string) =>
      currentBreakpoint && BREAKPOINTS[currentBreakpoint] >= BREAKPOINTS[breakpoint]
        ? className
        : '',

    below: (breakpoint: BreakpointKey, className: string) =>
      currentBreakpoint && BREAKPOINTS[currentBreakpoint] < BREAKPOINTS[breakpoint]
        ? className
        : '',
  };
}

// 响应式样式Hook
export function useResponsiveStyle() {
  const responsive = useResponsive();

  return {
    // 根据设备类型返回不同的样式
    device: <T>(styles: Partial<Record<DeviceSize, T>>, defaultStyle: T): T => {
      // 直接使用当前响应式状态而不是调用Hook
      const { isMobile, isTablet, isDesktop } = responsive;
      if (isMobile && styles.mobile) return styles.mobile;
      if (isTablet && styles.tablet) return styles.tablet;
      if (isDesktop && styles.desktop) return styles.desktop;
      return defaultStyle;
    },

    // 根据断点返回不同的样式
    breakpoint: <T>(styles: Partial<Record<ScreenSize, T>>, defaultStyle: T): T => {
      const { currentBreakpoint } = responsive;
      return styles[currentBreakpoint] || defaultStyle;
    },

    // 移动端样式
    mobile: <T>(mobileStyle: T, defaultStyle: T): T =>
      responsive.isMobile ? mobileStyle : defaultStyle,

    // 桌面端样式
    desktop: <T>(desktopStyle: T, defaultStyle: T): T =>
      responsive.isDesktop ? desktopStyle : defaultStyle,
  };
}

// 导出常用的响应式工具
export const responsive = {
  breakpoints: BREAKPOINTS,
  getCurrentBreakpoint,
  getDeviceType,

  // 媒体查询字符串生成器
  mediaQuery: {
    above: (breakpoint: BreakpointKey) => `(min-width: ${BREAKPOINTS[breakpoint]}px)`,
    below: (breakpoint: BreakpointKey) => `(max-width: ${BREAKPOINTS[breakpoint] - 1}px)`,
    between: (min: BreakpointKey, max: BreakpointKey) =>
      `(min-width: ${BREAKPOINTS[min]}px) and (max-width: ${BREAKPOINTS[max] - 1}px)`,
    mobile: `(max-width: ${BREAKPOINTS.md - 1}px)`,
    tablet: `(min-width: ${BREAKPOINTS.md}px) and (max-width: ${BREAKPOINTS.lg - 1}px)`,
    desktop: `(min-width: ${BREAKPOINTS.lg}px)`,
  },
};
