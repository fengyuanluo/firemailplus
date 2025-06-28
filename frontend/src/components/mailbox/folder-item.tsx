'use client';

import { useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
  Inbox,
  Send,
  FileText,
  Trash2,
  AlertTriangle,
  Folder as FolderIcon,
} from 'lucide-react';
import { Folder } from '@/types/email';
import { useMailboxStore, useContextMenuStore } from '@/lib/store';
import { FolderTree } from './folder-tree';

interface FolderItemProps {
  folder: Folder & { children?: Folder[] };
  level?: number;
}

export function FolderItem({ folder, level = 0 }: FolderItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const { selectedFolder, selectFolder, setEmails } = useMailboxStore();
  const { openMenu } = useContextMenuStore();

  // 获取文件夹图标
  const getFolderIcon = () => {
    const iconClass = 'w-4 h-4';

    switch (folder.type) {
      case 'inbox':
        return <Inbox className={iconClass} />;
      case 'sent':
        return <Send className={iconClass} />;
      case 'drafts':
        return <FileText className={iconClass} />;
      case 'trash':
        return <Trash2 className={iconClass} />;
      case 'spam':
        return <AlertTriangle className={iconClass} />;
      default:
        return <FolderIcon className={iconClass} />;
    }
  };

  // 处理文件夹点击
  const handleFolderClick = () => {
    selectFolder(folder);

    // 如果有子文件夹，切换展开状态
    if (folder.children && folder.children.length > 0) {
      setIsExpanded(!isExpanded);
    }

    // 选择文件夹后，邮件列表会自动重新加载（通过 EmailList 组件的 useEffect）
  };

  // 处理右键菜单
  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    openMenu(
      { x: e.clientX, y: e.clientY },
      {
        type: 'folder',
        id: folder.id,
        data: folder,
      }
    );
  };

  // 计算缩进
  const indentStyle = {
    paddingLeft: `${level * 16 + 8}px`,
  };

  const hasChildren = folder.children && folder.children.length > 0;

  return (
    <div className="space-y-1">
      {/* 文件夹项 */}
      <div
        onClick={handleFolderClick}
        onContextMenu={handleContextMenu}
        style={indentStyle}
        className={`
          flex items-center gap-2 p-2 rounded-md cursor-pointer transition-colors
          hover:bg-gray-100 dark:hover:bg-gray-700
          ${
            selectedFolder?.id === folder.id
              ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
              : 'text-gray-700 dark:text-gray-300'
          }
        `}
      >
        {/* 展开/折叠图标 */}
        <div className="flex-shrink-0 w-4 h-4">
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )
          ) : null}
        </div>

        {/* 文件夹图标 */}
        <div className="flex-shrink-0">{getFolderIcon()}</div>

        {/* 文件夹名称和统计 */}
        <div className="flex-1 min-w-0 flex items-center justify-between">
          <span className="text-sm truncate">{folder.display_name}</span>

          {/* 未读邮件数量 */}
          {folder.unread_emails > 0 && (
            <span className="flex-shrink-0 ml-2 inline-flex items-center justify-center px-2 py-1 text-xs font-medium text-white bg-blue-500 rounded-full min-w-[20px]">
              {folder.unread_emails > 99 ? '99+' : folder.unread_emails}
            </span>
          )}
        </div>
      </div>

      {/* 子文件夹 */}
      {hasChildren && isExpanded && <FolderTree folders={folder.children!} level={level + 1} />}
    </div>
  );
}
