'use client';

import { Reply, ReplyAll, Forward, Archive, Trash2, Star, MoreHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Email, parseEmailAddress, parseEmailAddresses, formatEmailAddress } from '@/types/email';
import { TranslateButton } from './translate-button';
import { LanguageCode } from '@/lib/translate';
import { useComposeStore, useMailboxStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

interface EmailHeaderProps {
  email: Email;
  onTranslate: (targetLang: LanguageCode) => void;
  isTranslating?: boolean;
  currentTranslationLang?: LanguageCode;
}

export function EmailHeader({
  email,
  onTranslate,
  isTranslating = false,
  currentTranslationLang,
}: EmailHeaderProps) {
  const { initializeReply, initializeReplyAll, initializeForward } = useComposeStore();
  const { removeEmail } = useMailboxStore();
  // 解析邮件地址
  const fromAddress =
    typeof email.from === 'string'
      ? parseEmailAddress(email.from)
      : Array.isArray(email.from)
        ? email.from[0]
        : null;
  const toAddresses =
    typeof email.to === 'string'
      ? parseEmailAddresses(email.to)
      : Array.isArray(email.to)
        ? email.to
        : [];
  const ccAddresses =
    typeof email.cc === 'string'
      ? parseEmailAddresses(email.cc || '')
      : Array.isArray(email.cc)
        ? email.cc
        : [];

  // 格式化时间
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      weekday: 'long',
    });
  };

  // 处理回复
  const handleReply = () => {
    initializeReply(email);
    toast.success('已打开回复窗口');
  };

  // 处理回复全部
  const handleReplyAll = () => {
    initializeReplyAll(email);
    toast.success('已打开回复全部窗口');
  };

  // 处理转发
  const handleForward = () => {
    initializeForward(email);
    toast.success('已打开转发窗口');
  };

  // 处理归档
  const handleArchive = async () => {
    try {
      await apiClient.archiveEmail(email.id);
      removeEmail(email.id);
      toast.success('邮件已归档');
    } catch (error: any) {
      toast.error(error.message || '归档失败');
    }
  };

  // 处理删除
  const handleDelete = async () => {
    if (confirm('确定要删除这封邮件吗？')) {
      try {
        await apiClient.deleteEmail(email.id);
        removeEmail(email.id);
        toast.success('邮件已删除');
      } catch (error: any) {
        toast.error(error.message || '删除失败');
      }
    }
  };

  // 格式化地址列表显示
  const formatAddressList = (addresses: any[], maxDisplay: number = 3) => {
    if (!addresses || addresses.length === 0) return '';

    const displayAddresses = addresses.slice(0, maxDisplay);
    const formatted = displayAddresses.map((addr) => formatEmailAddress(addr)).join(', ');

    if (addresses.length > maxDisplay) {
      return `${formatted} 等${addresses.length}人`;
    }

    return formatted;
  };

  return (
    <div className="flex-shrink-0 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
      <div className="p-6">
        {/* 第一行：发件人信息和功能按钮 */}
        <div className="flex items-start justify-between mb-4">
          {/* 左侧：发件人和收件人信息 */}
          <div className="flex-1 min-w-0 space-y-1">
            {/* 发件人 */}
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
                发件人:
              </span>
              <span className="text-sm text-gray-900 dark:text-gray-100 font-medium truncate">
                {fromAddress
                  ? formatEmailAddress(fromAddress)
                  : Array.isArray(email.from)
                    ? formatEmailAddress(email.from[0])
                    : email.from}
              </span>
            </div>

            {/* 收件人 */}
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
                收件人:
              </span>
              <span className="text-sm text-gray-700 dark:text-gray-300 truncate">
                {formatAddressList(toAddresses) || 'me@example.com'}
              </span>
            </div>

            {/* 抄送（如果有） */}
            {ccAddresses.length > 0 && (
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
                  抄送:
                </span>
                <span className="text-sm text-gray-700 dark:text-gray-300 truncate">
                  {formatAddressList(ccAddresses)}
                </span>
              </div>
            )}
          </div>

          {/* 右侧：功能按钮和时间 */}
          <div className="flex-shrink-0 text-right ml-4">
            {/* 第一行：功能按钮 */}
            <div className="flex items-center gap-1 mb-2">
              {/* 翻译按钮 */}
              <TranslateButton
                onTranslate={onTranslate}
                isTranslating={isTranslating}
                currentLang={currentTranslationLang}
                originalText={email.text_body || email.html_body}
              />

              {/* 回复按钮 */}
              <Button variant="ghost" size="sm" className="p-2 h-8" onClick={handleReply}>
                <Reply className="w-4 h-4" />
              </Button>

              {/* 回复全部按钮 */}
              <Button variant="ghost" size="sm" className="p-2 h-8" onClick={handleReplyAll}>
                <ReplyAll className="w-4 h-4" />
              </Button>

              {/* 转发按钮 */}
              <Button variant="ghost" size="sm" className="p-2 h-8" onClick={handleForward}>
                <Forward className="w-4 h-4" />
              </Button>

              {/* 归档按钮 */}
              <Button variant="ghost" size="sm" className="p-2 h-8" onClick={handleArchive}>
                <Archive className="w-4 h-4" />
              </Button>

              {/* 删除按钮 */}
              <Button variant="ghost" size="sm" className="p-2 h-8" onClick={handleDelete}>
                <Trash2 className="w-4 h-4" />
              </Button>

              {/* 星标按钮 */}
              <Button variant="ghost" size="sm" className="p-2 h-8">
                <Star
                  className={`w-4 h-4 ${email.is_starred ? 'text-yellow-500 fill-current' : ''}`}
                />
              </Button>
            </div>

            {/* 第二行：收件时间 */}
            <div className="text-xs text-gray-500 dark:text-gray-400">{formatDate(email.date)}</div>
          </div>
        </div>

        {/* 邮件主题 */}
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-0 line-clamp-2 break-words">
          {email.subject || '(无主题)'}
        </h1>
      </div>
    </div>
  );
}
