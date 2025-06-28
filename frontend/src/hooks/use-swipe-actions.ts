import { useState, useRef, useEffect } from 'react';

interface UseSwipeActionsOptions {
  itemId: number;
  isSwipeOpen: boolean;
  onSwipeStateChange: (itemId: number, isOpen: boolean) => void;
  swipeDistance?: number; // 最大滑动距离，默认120px
  threshold?: number; // 滑动阈值，默认-60px
}

export function useSwipeActions({
  itemId,
  isSwipeOpen,
  onSwipeStateChange,
  swipeDistance = 120,
  threshold = -60,
}: UseSwipeActionsOptions) {
  // 滑动状态
  const [isDragging, setIsDragging] = useState(false);
  const [startX, setStartX] = useState(0);
  const [currentX, setCurrentX] = useState(0);
  const [startY, setStartY] = useState(0);
  const [translateX, setTranslateX] = useState(0);

  // DOM引用
  const itemRef = useRef<HTMLDivElement>(null);
  const actionsRef = useRef<HTMLDivElement>(null);

  // 监听外部状态变化，同步UI
  useEffect(() => {
    if (isSwipeOpen) {
      setTranslateX(-swipeDistance);
    } else {
      setTranslateX(0);
    }
  }, [isSwipeOpen, swipeDistance]);

  // 处理触摸开始
  const handleTouchStart = (e: React.TouchEvent) => {
    const touch = e.touches[0];
    setStartX(touch.clientX);
    setCurrentX(touch.clientX);
    setStartY(touch.clientY);
    setIsDragging(true);
  };

  // 处理触摸移动
  const handleTouchMove = (e: React.TouchEvent) => {
    if (!isDragging) return;

    const touch = e.touches[0];
    setCurrentX(touch.clientX);

    const deltaX = touch.clientX - startX;
    const deltaY = Math.abs(touch.clientY - startY);

    // 如果垂直滑动距离大于水平滑动距离，则认为是垂直滚动，不处理
    if (deltaY > Math.abs(deltaX)) {
      setIsDragging(false);
      return;
    }

    // 只允许向左滑动
    if (deltaX < 0) {
      const newTranslateX = Math.max(deltaX, -swipeDistance);
      setTranslateX(newTranslateX);
      // 尝试防止页面滚动，但不强制调用preventDefault以避免passive listener错误
      try {
        e.preventDefault();
      } catch (error) {
        // 忽略passive listener错误，依赖CSS touch-action属性控制
        console.debug('preventDefault failed in passive listener, relying on CSS touch-action');
      }
    }
  };

  // 处理触摸结束
  const handleTouchEnd = () => {
    if (!isDragging) return;

    const deltaX = currentX - startX;

    if (deltaX < threshold) {
      // 滑动距离超过阈值，显示操作按钮
      onSwipeStateChange(itemId, true);
      setTranslateX(-swipeDistance);
    } else {
      // 滑动距离不够，回弹
      onSwipeStateChange(itemId, false);
      setTranslateX(0);
    }

    setIsDragging(false);
  };

  // 处理鼠标事件（用于桌面端测试）
  const handleMouseDown = (e: React.MouseEvent) => {
    // 只在非触摸设备上处理鼠标事件
    if ('ontouchstart' in window) return;

    e.preventDefault();
    setStartX(e.clientX);
    setCurrentX(e.clientX);
    setStartY(e.clientY);
    setIsDragging(true);
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging || 'ontouchstart' in window) return;

    setCurrentX(e.clientX);
    const deltaX = e.clientX - startX;

    if (deltaX < 0) {
      const newTranslateX = Math.max(deltaX, -swipeDistance);
      setTranslateX(newTranslateX);
    }
  };

  const handleMouseUp = () => {
    if (!isDragging || 'ontouchstart' in window) return;

    const deltaX = currentX - startX;

    if (deltaX < threshold) {
      onSwipeStateChange(itemId, true);
      setTranslateX(-swipeDistance);
    } else {
      onSwipeStateChange(itemId, false);
      setTranslateX(0);
    }

    setIsDragging(false);
  };

  // 关闭滑动菜单
  const closeSwipe = () => {
    onSwipeStateChange(itemId, false);
    setTranslateX(0);
  };

  return {
    itemRef,
    actionsRef,
    isDragging,
    translateX,
    handleTouchStart,
    handleTouchMove,
    handleTouchEnd,
    handleMouseDown,
    handleMouseMove,
    handleMouseUp,
    closeSwipe,
  };
}
