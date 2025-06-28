'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Folder, FolderOpen, Inbox, Send, FileText, Trash2, AlertTriangle } from 'lucide-react';
import { useMailboxStore } from '@/lib/store';
import {
  MobileLayout,
  MobilePage,
  MobileContent,
  MobileList,
  MobileListItem,
  MobileEmptyState,
  MobileLoading,
} from './mobile-layout';
import { FoldersHeader } from './mobile-header';
import { apiClient } from '@/lib/api';

interface MobileFoldersPageProps {
  accountId: number;
}

export function MobileFoldersPage({ accountId }: MobileFoldersPageProps) {
  const { accounts, folders, selectedFolder, selectFolder, setFolders } = useMailboxStore();

  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // 获取当前账户
  const currentAccount = accounts.find((account) => account.id === accountId);

  // 加载文件夹列表
  useEffect(() => {
    const loadFolders = async () => {
      if (!currentAccount) return;

      setIsLoading(true);
      try {
        const response = await apiClient.getFolders(accountId);
        if (response.success && response.data) {
          setFolders(response.data);
        }
      } catch (error) {
        console.error('Failed to load folders:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadFolders();
  }, [accountId, currentAccount, setFolders]);

  // 处理文件夹选择
  const handleFolderSelect = (folder: any) => {
    selectFolder(folder);
    router.push(`/mailbox/mobile/folder/${folder.id}`);
  };

  // 获取文件夹图标
  const getFolderIcon = (folderType: string) => {
    switch (folderType) {
      case 'inbox':
        return <Inbox className="w-5 h-5" />;
      case 'sent':
        return <Send className="w-5 h-5" />;
      case 'drafts':
        return <FileText className="w-5 h-5" />;
      case 'trash':
        return <Trash2 className="w-5 h-5" />;
      case 'spam':
        return <AlertTriangle className="w-5 h-5" />;
      default:
        return <Folder className="w-5 h-5" />;
    }
  };

  // 获取文件夹颜色
  const getFolderColor = (folderType: string) => {
    switch (folderType) {
      case 'inbox':
        return 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900';
      case 'sent':
        return 'text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900';
      case 'drafts':
        return 'text-yellow-600 dark:text-yellow-400 bg-yellow-100 dark:bg-yellow-900';
      case 'trash':
        return 'text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900';
      case 'spam':
        return 'text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-900';
      default:
        return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700';
    }
  };

  if (!currentAccount) {
    return (
      <MobileLayout>
        <MobilePage>
          <FoldersHeader accountName="未知账户" />
          <MobileContent>
            <MobileEmptyState
              icon={<Folder className="w-8 h-8 text-gray-400" />}
              title="账户不存在"
              description="请返回选择有效的邮箱账户"
            />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  if (isLoading) {
    return (
      <MobileLayout>
        <MobilePage>
          <FoldersHeader accountName={currentAccount.name} />
          <MobileContent>
            <MobileLoading message="加载文件夹..." />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  return (
    <MobileLayout>
      <MobilePage>
        <FoldersHeader accountName={currentAccount.name} />

        <MobileContent padding={false}>
          {folders.length > 0 ? (
            <MobileList>
              {folders.map((folder) => (
                <MobileListItem
                  key={folder.id}
                  onClick={() => handleFolderSelect(folder)}
                  active={selectedFolder?.id === folder.id}
                >
                  <div className="flex items-center gap-3">
                    {/* 文件夹图标 */}
                    <div
                      className={`w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 ${getFolderColor(folder.type)}`}
                    >
                      {getFolderIcon(folder.type)}
                    </div>

                    {/* 文件夹信息 */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <h3 className="text-base font-medium text-gray-900 dark:text-gray-100 truncate">
                          {folder.display_name || folder.name}
                        </h3>

                        <div className="flex items-center gap-2">
                          {folder.unread_emails > 0 && (
                            <span className="bg-blue-600 text-white text-xs px-2 py-1 rounded-full min-w-[20px] text-center">
                              {folder.unread_emails > 99 ? '99+' : folder.unread_emails}
                            </span>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-4 mt-1">
                        <span className="text-sm text-gray-500 dark:text-gray-400">
                          {folder.total_emails} 封邮件
                        </span>

                        {folder.unread_emails > 0 && (
                          <span className="text-sm text-blue-600 dark:text-blue-400">
                            {folder.unread_emails} 封未读
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </MobileListItem>
              ))}
            </MobileList>
          ) : (
            <MobileEmptyState
              icon={<Folder className="w-8 h-8 text-gray-400" />}
              title="暂无文件夹"
              description="此邮箱账户暂无可用文件夹"
            />
          )}
        </MobileContent>
      </MobilePage>
    </MobileLayout>
  );
}

// 文件夹卡片组件
interface FolderCardProps {
  folder: any;
  isSelected: boolean;
  onClick: () => void;
}

function FolderCard({ folder, isSelected, onClick }: FolderCardProps) {
  const getFolderIcon = (folderType: string) => {
    switch (folderType) {
      case 'inbox':
        return <Inbox className="w-6 h-6" />;
      case 'sent':
        return <Send className="w-6 h-6" />;
      case 'drafts':
        return <FileText className="w-6 h-6" />;
      case 'trash':
        return <Trash2 className="w-6 h-6" />;
      case 'spam':
        return <AlertTriangle className="w-6 h-6" />;
      default:
        return <Folder className="w-6 h-6" />;
    }
  };

  return (
    <div
      onClick={onClick}
      className={`
        p-4 rounded-lg border cursor-pointer transition-all duration-150
        ${
          isSelected
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
        }
      `}
    >
      <div className="flex items-center gap-3">
        <div className="w-12 h-12 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center flex-shrink-0">
          {getFolderIcon(folder.type)}
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-1">
            <h3 className="font-medium text-gray-900 dark:text-gray-100 truncate">
              {folder.display_name || folder.name}
            </h3>
            {folder.unread_emails > 0 && (
              <span className="bg-blue-600 text-white text-xs px-2 py-1 rounded-full">
                {folder.unread_emails}
              </span>
            )}
          </div>

          <div className="flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
            <span>{folder.total_emails} 封邮件</span>
            {folder.unread_emails > 0 && (
              <span className="text-blue-600 dark:text-blue-400">
                {folder.unread_emails} 封未读
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
