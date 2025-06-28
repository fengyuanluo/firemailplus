'use client';

import { useEffect, useState, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { Mail, Star, Paperclip, MoreVertical, Reply, Archive, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMailboxStore, useComposeStore } from '@/lib/store';
import {
  MobileLayout,
  MobilePage,
  MobileContent,
  MobileList,
  MobileListItem,
  MobileEmptyState,
  MobileLoading,
} from './mobile-layout';
import { EmailsHeader } from './mobile-header';
import { apiClient } from '@/lib/api';
import { parseEmailAddress, formatEmailAddress, getEmailPreview, type Email } from '@/types/email';
import { toast } from 'sonner';
import { useMobileNavigation } from '@/hooks/use-mobile-navigation';
import { useSwipeActions } from '@/hooks/use-swipe-actions';

interface MobileEmailsPageProps {
  folderId: number;
}

export function MobileEmailsPage({ folderId }: MobileEmailsPageProps) {
  const { folders, emails, selectedEmail, selectEmail, setEmails } = useMailboxStore();

  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [emailList, setEmailList] = useState<Email[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [openSwipeEmailId, setOpenSwipeEmailId] = useState<number | null>(null);
  const router = useRouter();

  // 获取当前文件夹
  const currentFolder = folders.find((folder) => folder.id === folderId);

  // 加载邮件列表
  const loadEmails = async (page: number = 1, append: boolean = false) => {
    if (page === 1) {
      setIsLoading(true);
    } else {
      setIsLoadingMore(true);
    }

    try {
      const response = await apiClient.getEmails({
        folder_id: folderId,
        page: page,
        page_size: 50,
      });
      if (response.success && response.data) {
        const newEmails = response.data.emails;
        if (append) {
          setEmailList((prev) => [...prev, ...newEmails]);
        } else {
          setEmailList(newEmails);
          setEmails(newEmails);
        }

        // 检查是否还有更多数据
        const totalPages = Math.ceil(response.data.total / response.data.page_size);
        setHasMore(page < totalPages);
      }
    } catch (error) {
      console.error('Failed to load emails:', error);
    } finally {
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  useEffect(() => {
    if (currentFolder) {
      setCurrentPage(1);
      setHasMore(true);
      loadEmails(1, false);
    }
  }, [folderId, currentFolder]);

  // 处理邮件选择
  const handleEmailSelect = (email: Email) => {
    selectEmail(email);
    router.push(`/mailbox/mobile/email/${email.id}`);
  };

  // 处理滚动加载更多
  const handleLoadMore = () => {
    if (!isLoadingMore && hasMore) {
      const nextPage = currentPage + 1;
      setCurrentPage(nextPage);
      loadEmails(nextPage, true);
    }
  };

  // 处理滑动状态变化
  const handleSwipeStateChange = (emailId: number, isOpen: boolean) => {
    if (isOpen) {
      // 如果要打开新的菜单，先关闭其他所有菜单
      setOpenSwipeEmailId(emailId);
    } else {
      // 关闭菜单
      if (openSwipeEmailId === emailId) {
        setOpenSwipeEmailId(null);
      }
    }
  };

  if (!currentFolder) {
    return (
      <MobileLayout>
        <MobilePage>
          <EmailsHeader folderName="未知文件夹" />
          <MobileContent>
            <MobileEmptyState
              icon={<Mail className="w-8 h-8 text-gray-400" />}
              title="文件夹不存在"
              description="请返回选择有效的文件夹"
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
          <EmailsHeader folderName={currentFolder.display_name || currentFolder.name} />
          <MobileContent>
            <MobileLoading message="加载邮件列表..." />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  return (
    <MobileLayout>
      <MobilePage>
        <EmailsHeader folderName={currentFolder.display_name || currentFolder.name} />

        <MobileContent padding={false}>
          {emailList.length > 0 ? (
            <div>
              <MobileList>
                {emailList.map((email: Email) => (
                  <MobileEmailItem
                    key={email.id}
                    email={email}
                    isSelected={selectedEmail?.id === email.id}
                    onClick={() => handleEmailSelect(email)}
                    isSwipeOpen={openSwipeEmailId === email.id}
                    onSwipeStateChange={handleSwipeStateChange}
                  />
                ))}
              </MobileList>

              {/* 加载更多按钮 */}
              {hasMore && (
                <div className="p-4 text-center">
                  <button
                    onClick={handleLoadMore}
                    disabled={isLoadingMore}
                    className="w-full py-2 px-4 bg-blue-500 text-white rounded-lg disabled:bg-gray-300 disabled:cursor-not-allowed"
                  >
                    {isLoadingMore ? '加载中...' : '加载更多'}
                  </button>
                </div>
              )}
            </div>
          ) : (
            <MobileEmptyState
              icon={<Mail className="w-8 h-8 text-gray-400" />}
              title="暂无邮件"
              description="此文件夹中暂无邮件"
            />
          )}
        </MobileContent>
      </MobilePage>
    </MobileLayout>
  );
}

// 移动端邮件项组件
interface MobileEmailItemProps {
  email: Email;
  isSelected: boolean;
  onClick: () => void;
  isSwipeOpen: boolean;
  onSwipeStateChange: (emailId: number, isOpen: boolean) => void;
}

function MobileEmailItem({
  email,
  isSelected,
  onClick,
  isSwipeOpen,
  onSwipeStateChange,
}: MobileEmailItemProps) {
  const { updateEmail, removeEmail } = useMailboxStore();
  const { initializeReply } = useComposeStore();
  const { navigateToCompose } = useMobileNavigation();

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
    itemId: email.id,
    isSwipeOpen,
    onSwipeStateChange,
  });

  // 邮件操作函数
  const handleReply = async () => {
    try {
      initializeReply(email);
      navigateToCompose({ reply: email.id.toString() });
      closeSwipe();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    }
  };

  const handleToggleStar = async () => {
    try {
      await apiClient.toggleEmailStar(email.id);
      updateEmail(email.id, { is_starred: !email.is_starred });
      toast.success(email.is_starred ? '已移除星标' : '已添加星标');
      closeSwipe();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    }
  };

  const handleArchive = async () => {
    try {
      await apiClient.archiveEmail(email.id);
      removeEmail(email.id);
      toast.success('邮件已归档');
      closeSwipe();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '归档失败');
    }
  };

  const handleDelete = async () => {
    try {
      await apiClient.deleteEmail(email.id);
      removeEmail(email.id);
      toast.success('邮件已删除');
      closeSwipe();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    }
  };

  // 处理邮件项点击
  const handleItemClick = () => {
    if (isSwipeOpen) {
      // 如果菜单是打开的，先关闭菜单
      closeSwipe();
    } else {
      // 否则执行正常的点击操作
      onClick();
    }
  };

  // 解析发件人信息
  const fromAddress =
    typeof email.from === 'string'
      ? parseEmailAddress(email.from)
      : Array.isArray(email.from)
        ? email.from[0]
        : null;
  const fromDisplay =
    fromAddress?.name ||
    fromAddress?.address ||
    (Array.isArray(email.from) ? email.from[0]?.address : email.from);

  // 获取邮件预览
  const preview = getEmailPreview(email);

  // 格式化时间
  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) {
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      });
    } else if (days === 1) {
      return '昨天';
    } else if (days < 7) {
      return `${days}天前`;
    } else {
      return date.toLocaleDateString('zh-CN', {
        month: 'short',
        day: 'numeric',
      });
    }
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
        <MobileListItem
          onClick={handleItemClick}
          active={isSelected}
          className={`${!email.is_read ? 'bg-blue-50 dark:bg-blue-900/10' : ''}`}
        >
          <div className="flex gap-3">
            {/* 头像或图标 */}
            <div className="w-10 h-10 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center flex-shrink-0">
              <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                {fromDisplay ? fromDisplay.charAt(0).toUpperCase() : 'U'}
              </span>
            </div>

            {/* 邮件信息 */}
            <div className="flex-1 min-w-0">
              {/* 第一行：发件人和时间 */}
              <div className="flex items-center justify-between mb-1">
                <span
                  className={`text-sm truncate ${
                    !email.is_read
                      ? 'font-semibold text-gray-900 dark:text-gray-100'
                      : 'font-medium text-gray-700 dark:text-gray-300'
                  }`}
                >
                  {fromDisplay}
                </span>

                <div className="flex items-center gap-1 flex-shrink-0 ml-2">
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {formatTime(email.date)}
                  </span>

                  {/* 状态图标 */}
                  <div className="flex items-center gap-1">
                    {email.is_starred && <Star className="w-3 h-3 text-yellow-500 fill-current" />}
                    {email.has_attachment && <Paperclip className="w-3 h-3 text-gray-400" />}
                    {!email.is_read && <div className="w-2 h-2 bg-blue-600 rounded-full" />}
                  </div>
                </div>
              </div>

              {/* 第二行：主题 */}
              <div
                className={`text-sm mb-1 truncate ${
                  !email.is_read
                    ? 'font-medium text-gray-900 dark:text-gray-100'
                    : 'text-gray-700 dark:text-gray-300'
                }`}
              >
                {email.subject || '(无主题)'}
              </div>

              {/* 第三行：预览 */}
              {preview && (
                <div className="text-xs text-gray-500 dark:text-gray-400 line-clamp-2">
                  {preview}
                </div>
              )}
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
          onClick={handleReply}
          className="h-full w-7.5 rounded-none bg-blue-500 hover:bg-blue-600 text-white flex flex-col items-center justify-center"
          style={{ width: '30px' }}
        >
          <Reply className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggleStar}
          className={`h-full rounded-none ${
            email.is_starred ? 'bg-yellow-500 hover:bg-yellow-600' : 'bg-gray-500 hover:bg-gray-600'
          } text-white flex flex-col items-center justify-center`}
          style={{ width: '30px' }}
        >
          <Star className={`w-4 h-4 ${email.is_starred ? 'fill-current' : ''}`} />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleArchive}
          className="h-full rounded-none bg-green-500 hover:bg-green-600 text-white flex flex-col items-center justify-center"
          style={{ width: '30px' }}
        >
          <Archive className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleDelete}
          className="h-full rounded-none bg-red-500 hover:bg-red-600 text-white flex flex-col items-center justify-center"
          style={{ width: '30px' }}
        >
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
