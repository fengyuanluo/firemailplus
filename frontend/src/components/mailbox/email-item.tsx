'use client';

import { useState, forwardRef } from 'react';
import { Star, Paperclip, Circle } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Email, parseEmailAddress, getEmailPreview, formatFileSize } from '@/types/email';
import { useMailboxStore, useContextMenuStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

interface EmailItemProps {
  email: Email;
  isSelected?: boolean;
  onClick?: () => void;
}

export const EmailItem = forwardRef<HTMLDivElement, EmailItemProps>(function EmailItem(
  { email, isSelected: externalIsSelected, onClick },
  ref
) {
  const [isHovered, setIsHovered] = useState(false);

  const { selectedEmail, selectedEmails, selectEmail, toggleEmailSelection, updateEmail } =
    useMailboxStore();

  const isEmailSelected = (emailId: number) => selectedEmails.has(emailId);

  const { openMenu } = useContextMenuStore();

  const isSelected =
    externalIsSelected !== undefined ? externalIsSelected : selectedEmail?.id === email.id;
  const isChecked = isEmailSelected(email.id);

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

  // 生成邮件预览
  const preview = getEmailPreview(email, 100);

  // 格式化时间
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffTime = Math.abs(now.getTime() - date.getTime());
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays === 1) {
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      });
    } else if (diffDays <= 7) {
      return date.toLocaleDateString('zh-CN', {
        month: 'short',
        day: 'numeric',
      });
    } else {
      return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    }
  };

  // 处理邮件点击
  const handleEmailClick = (e: React.MouseEvent) => {
    // 如果点击的是复选框或星标按钮，不处理邮件选择
    if ((e.target as HTMLElement).closest('[data-action]')) {
      return;
    }

    if (onClick) {
      // 如果提供了外部点击处理函数，使用它
      onClick();
    } else {
      // 否则使用默认行为
      selectEmail(email);
    }

    // 如果邮件未读，标记为已读
    if (!email.is_read) {
      handleMarkAsRead();
    }
  };

  // 处理复选框点击
  const handleCheckboxChange = () => {
    toggleEmailSelection(email.id);
  };

  // 处理星标点击
  const handleStarClick = async () => {
    try {
      const response = await apiClient.toggleEmailStar(email.id);
      if (response.success) {
        updateEmail(email.id, { is_starred: !email.is_starred });
      }
    } catch (error: any) {
      toast.error(error.message || '操作失败');
    }
  };

  // 处理标记已读
  const handleMarkAsRead = async () => {
    if (email.is_read) return;

    try {
      const response = await apiClient.markEmailAsRead(email.id);
      if (response.success) {
        updateEmail(email.id, { is_read: true });
      }
    } catch (error: any) {
      toast.error(error.message || '标记已读失败');
    }
  };

  // 处理右键菜单
  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    openMenu(
      { x: e.clientX, y: e.clientY },
      {
        type: 'email',
        id: email.id,
        data: email,
      }
    );
  };

  return (
    <div
      ref={ref}
      onClick={handleEmailClick}
      onContextMenu={handleContextMenu}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      className={`
        relative p-4 cursor-pointer transition-colors duration-150 border-b border-gray-100 dark:border-gray-700
        hover:bg-gray-50 dark:hover:bg-gray-750
        ${isSelected ? 'bg-blue-50 dark:bg-blue-900/20 border-r-2 border-blue-500' : ''}
        ${!email.is_read ? 'bg-white dark:bg-gray-800' : 'bg-gray-50 dark:bg-gray-750'}
      `}
    >
      {/* 未读指示器 */}
      {!email.is_read && (
        <div className="absolute left-2 top-1/2 transform -translate-y-1/2 w-2 h-2 bg-blue-500 rounded-full"></div>
      )}

      <div className="flex items-start gap-3 ml-4">
        {/* 复选框 */}
        <div
          className={`flex-shrink-0 transition-opacity ${isHovered || isChecked ? 'opacity-100' : 'opacity-0'}`}
          data-action="checkbox"
        >
          <Checkbox
            checked={isChecked}
            onCheckedChange={handleCheckboxChange}
            className="data-[state=checked]:bg-blue-600 data-[state=checked]:border-blue-600"
          />
        </div>

        {/* 邮件内容 */}
        <div className="flex-1 min-w-0 space-y-1">
          {/* 第一行：发件人和时间 */}
          <div className="flex items-center justify-between gap-2">
            <span
              className={`
              text-sm truncate flex-1
              ${
                !email.is_read
                  ? 'font-semibold text-gray-900 dark:text-gray-100'
                  : 'font-normal text-gray-700 dark:text-gray-300'
              }
            `}
            >
              {fromDisplay}
            </span>

            <div className="flex items-center gap-1 flex-shrink-0">
              {/* 附件图标 */}
              {email.has_attachment && <Paperclip className="w-3 h-3 text-gray-400" />}

              {/* 时间 */}
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {formatDate(email.date)}
              </span>
            </div>
          </div>

          {/* 第二行：主题 */}
          <div className="flex items-center justify-between gap-2">
            <h3
              className={`
              text-sm line-clamp-1 flex-1
              ${
                !email.is_read
                  ? 'font-semibold text-gray-900 dark:text-gray-100'
                  : 'font-normal text-gray-700 dark:text-gray-300'
              }
            `}
            >
              {email.subject || '(无主题)'}
            </h3>
          </div>

          {/* 第三行：邮件预览 */}
          <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-1">{preview}</p>
        </div>

        {/* 右侧操作区域 */}
        <div className="flex-shrink-0 flex items-center gap-1">
          {/* 星标按钮 */}
          <Button
            variant="ghost"
            size="sm"
            onClick={handleStarClick}
            className={`
              p-1 h-auto transition-opacity
              ${isHovered || email.is_starred ? 'opacity-100' : 'opacity-0'}
              ${email.is_starred ? 'text-yellow-500' : 'text-gray-400 hover:text-yellow-500'}
            `}
            data-action="star"
          >
            <Star className={`w-4 h-4 ${email.is_starred ? 'fill-current' : ''}`} />
          </Button>

          {/* 重要性指示器 */}
          {email.is_important && <Circle className="w-3 h-3 text-red-500 fill-current" />}
        </div>
      </div>
    </div>
  );
});
