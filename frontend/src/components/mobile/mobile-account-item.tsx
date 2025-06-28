'use client';

import { useState, useRef, useEffect } from 'react';
import { Settings, Trash2, Circle, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { MobileListItem } from './mobile-layout';
import { useSwipeActions } from '@/hooks/use-swipe-actions';
import type { EmailAccount } from '@/types/email';

interface MobileAccountItemProps {
  account: EmailAccount;
  onClick: () => void;
  onSettings: (account: EmailAccount) => void;
  onDelete: (account: EmailAccount) => void;
  active?: boolean;
}

export function MobileAccountItem({
  account,
  onClick,
  onSettings,
  onDelete,
  active = false,
}: MobileAccountItemProps) {
  const [isSwipeOpen, setIsSwipeOpen] = useState(false);

  // 使用滑动操作hook
  const {
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
  } = useSwipeActions({
    itemId: account.id,
    isSwipeOpen,
    onSwipeStateChange: (_, isOpen) => setIsSwipeOpen(isOpen),
  });

  // 获取状态指示器
  const getStatusIndicator = () => {
    switch (account.sync_status) {
      case 'syncing':
        return <Loader2 className="w-3 h-3 text-blue-500 animate-spin" />;
      case 'error':
        return <AlertCircle className="w-3 h-3 text-red-500" />;
      case 'success':
        return <Circle className="w-3 h-3 text-green-500 fill-current" />;
      default:
        return <Circle className="w-3 h-3 text-gray-400" />;
    }
  };

  // 点击外部关闭滑动菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent | TouchEvent) => {
      if (
        isSwipeOpen &&
        itemRef.current &&
        !itemRef.current.contains(event.target as Node) &&
        actionsRef.current &&
        !actionsRef.current.contains(event.target as Node)
      ) {
        closeSwipe();
      }
    };

    if (isSwipeOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('touchstart', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('touchstart', handleClickOutside);
    };
  }, [isSwipeOpen, closeSwipe]);

  const handleItemClick = () => {
    if (isSwipeOpen) {
      closeSwipe();
    } else {
      onClick();
    }
  };

  const handleSettings = (e: React.MouseEvent) => {
    e.stopPropagation();
    onSettings(account);
    closeSwipe();
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDelete(account);
    closeSwipe();
  };

  return (
    <div className="relative overflow-hidden">
      {/* 主要内容 */}
      <div
        ref={itemRef}
        className={`relative z-10 transition-transform duration-200 ease-out bg-white dark:bg-gray-800 ${
          isDragging ? 'transition-none' : ''
        }`}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        style={{
          touchAction: 'pan-y',
          transform: `translateX(${translateX}px)`,
          width: '100%',
        }}
      >
        <MobileListItem onClick={handleItemClick} active={active}>
          <div className="flex items-center gap-3">
            {/* 账户图标 */}
            <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/20 rounded-full flex items-center justify-center flex-shrink-0">
              <span className="text-sm font-medium text-blue-600 dark:text-blue-400">
                {account.name.charAt(0).toUpperCase()}
              </span>
            </div>

            {/* 账户信息 */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <span className="font-medium text-gray-900 dark:text-gray-100 truncate">
                  {account.name}
                </span>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {getStatusIndicator()}
                  {account.unread_emails > 0 && (
                    <span className="bg-blue-600 text-white text-xs px-2 py-1 rounded-full min-w-[20px] text-center">
                      {account.unread_emails > 99 ? '99+' : account.unread_emails}
                    </span>
                  )}
                </div>
              </div>
              <div className="text-sm text-gray-500 dark:text-gray-400 truncate">
                {account.email}
              </div>
            </div>
          </div>
        </MobileListItem>
      </div>

      {/* 滑动操作按钮 */}
      <div
        ref={actionsRef}
        className="absolute top-0 right-0 h-full flex items-center z-0"
        style={{ width: '120px' }}
      >
        <Button
          variant="ghost"
          size="sm"
          onClick={handleSettings}
          className="h-full w-16 rounded-none bg-blue-500 hover:bg-blue-600 text-white"
        >
          <Settings className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleDelete}
          className="h-full w-16 rounded-none bg-red-500 hover:bg-red-600 text-white"
        >
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
