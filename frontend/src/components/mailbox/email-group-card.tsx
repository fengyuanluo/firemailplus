'use client';

import type { ReactNode, DragEvent, MouseEvent } from 'react';
import { Button } from '@/components/ui/button';
import type { EmailGroup } from '@/types/email';
import { CheckSquare, FolderPlus, Hash } from 'lucide-react';

interface EmailGroupCardProps {
  group: EmailGroup;
  accountsCount: number;
  isDropTarget?: boolean;
  collapsed?: boolean;
  selectionMode?: boolean;
  allSelected?: boolean;
  onToggleCollapse?: () => void;
  onToggleGroupSelection?: () => void;
  showHandle?: boolean;
  handleText?: string;
  draggable?: boolean;
  onDragStart?: (e: DragEvent) => void;
  onDragEnd?: (e: DragEvent) => void;
  onDragOver?: (e: DragEvent) => void;
  onDrop?: (e: DragEvent) => void;
  onContextMenu?: (e: MouseEvent) => void;
  bodyClassName?: string;
  className?: string;
  children?: ReactNode;
  subtitleSuffix?: string;
}

export function EmailGroupCard({
  group,
  accountsCount,
  isDropTarget = false,
  collapsed = false,
  selectionMode = false,
  allSelected = false,
  onToggleCollapse,
  onToggleGroupSelection,
  showHandle = true,
  handleText,
  draggable = true,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDrop,
  onContextMenu,
  bodyClassName = 'mt-2 space-y-2',
  className = '',
  children,
  subtitleSuffix,
}: EmailGroupCardProps) {
  const resolvedHandleText = handleText ?? (group.is_default ? '默认分组' : '拖动排序');
  const resolvedSubtitle = subtitleSuffix ?? (group.is_default ? '默认' : '可拖动排序');
  const containerClasses = `border rounded-lg p-3 transition-colors ${
    isDropTarget ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-gray-200 dark:border-gray-700'
  } ${className}`;

  return (
    <div
      className={containerClasses}
      onDragOver={(e) => {
        if (onDragOver) {
          e.preventDefault();
          onDragOver(e);
        }
      }}
      onDrop={(e) => {
        if (onDrop) {
          e.preventDefault();
          onDrop(e);
        }
      }}
      onDragEnd={onDragEnd}
      onContextMenu={onContextMenu}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <div
            className="flex items-center justify-center w-8 h-8 rounded-full bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300 cursor-pointer"
            onClick={(e) => {
              e.stopPropagation();
              onToggleCollapse?.();
            }}
          >
            {group.is_default ? <FolderPlus className="w-4 h-4" /> : <Hash className="w-4 h-4" />}
          </div>
          <div>
            <div className="text-sm font-semibold text-gray-900 dark:text-gray-100">{group.name}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {accountsCount} 个邮箱
              {resolvedSubtitle ? ` · ${resolvedSubtitle}` : ''}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {selectionMode && accountsCount > 0 && onToggleGroupSelection && (
            <Button
              size="icon"
              variant="ghost"
              className="h-8 w-8"
              onClick={(e) => {
                e.stopPropagation();
                onToggleGroupSelection();
              }}
              title="选择/取消选择此分组全部邮箱"
            >
              <CheckSquare className={`w-4 h-4 ${allSelected ? 'text-blue-500' : 'text-gray-400'}`} />
            </Button>
          )}

          {showHandle && (
            <div
              className={`text-xs text-gray-400 ${group.is_default || !draggable ? '' : 'cursor-grab'}`}
              draggable={!group.is_default && draggable}
              onDragStart={group.is_default || !draggable ? undefined : onDragStart}
              onDragEnd={onDragEnd}
            >
              {resolvedHandleText}
            </div>
          )}
        </div>
      </div>

      {!collapsed && children && <div className={bodyClassName}>{children}</div>}
    </div>
  );
}
