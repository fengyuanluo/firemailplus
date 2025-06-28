'use client';

import { useMemo } from 'react';
import { Folder } from '@/types/email';
import { FolderItem } from './folder-item';

interface FolderTreeProps {
  folders: Folder[];
  level?: number;
}

export function FolderTree({ folders, level = 0 }: FolderTreeProps) {
  // 构建文件夹树形结构
  const folderTree = useMemo(() => {
    // 创建文件夹映射
    const folderMap = new Map<number, Folder & { children: Folder[] }>();

    // 初始化所有文件夹
    folders.forEach((folder) => {
      folderMap.set(folder.id, { ...folder, children: [] });
    });

    // 构建父子关系
    const rootFolders: (Folder & { children: Folder[] })[] = [];

    folders.forEach((folder) => {
      const folderWithChildren = folderMap.get(folder.id)!;

      if (folder.parent_id && folderMap.has(folder.parent_id)) {
        // 有父文件夹，添加到父文件夹的children中
        const parent = folderMap.get(folder.parent_id)!;
        parent.children.push(folderWithChildren);
      } else {
        // 没有父文件夹，是根文件夹
        rootFolders.push(folderWithChildren);
      }
    });

    // 按文件夹类型和名称排序
    const sortFolders = (folders: (Folder & { children: Folder[] })[]) => {
      return folders.sort((a, b) => {
        // 系统文件夹优先级
        const systemFolderOrder: Record<string, number> = {
          inbox: 1,
          sent: 2,
          drafts: 3,
          spam: 4,
          trash: 5,
        };

        const aOrder = systemFolderOrder[a.type] || 999;
        const bOrder = systemFolderOrder[b.type] || 999;

        if (aOrder !== bOrder) {
          return aOrder - bOrder;
        }

        // 同类型按名称排序
        return a.display_name.localeCompare(b.display_name);
      });
    };

    // 递归排序所有层级
    const sortRecursively = (folders: (Folder & { children: Folder[] })[]) => {
      const sorted = sortFolders(folders);
      sorted.forEach((folder) => {
        if (folder.children && folder.children.length > 0) {
          folder.children = sortRecursively(folder.children as (Folder & { children: Folder[] })[]);
        }
      });
      return sorted;
    };

    return sortRecursively(rootFolders);
  }, [folders]);

  if (folderTree.length === 0) {
    return <div className="p-2 text-sm text-gray-500 dark:text-gray-400">暂无文件夹</div>;
  }

  return (
    <div className="space-y-1">
      {folderTree.map((folder) => (
        <FolderItem key={folder.id} folder={folder} level={level} />
      ))}
    </div>
  );
}
