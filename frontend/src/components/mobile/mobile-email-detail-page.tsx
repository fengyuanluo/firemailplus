'use client';

import { useEffect, useState } from 'react';
import { useMobileNavigation } from '@/hooks/use-mobile-navigation';
import { ChevronDown, ChevronUp, Reply, Forward, Star, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMailboxStore } from '@/lib/store';
import {
  MobileLayout,
  MobilePage,
  MobileContent,
  MobileEmptyState,
  MobileLoading,
} from './mobile-layout';
import { EmailDetailHeader } from './mobile-header';
import { EmailDetail } from '@/components/mailbox/email-detail';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { LanguageCode } from '@/lib/translate';
import {
  formatEmailAddress,
  parseEmailAddress,
  parseEmailAddresses,
  type Email,
  type EmailAddress,
} from '@/types/email';

interface MobileEmailDetailPageProps {
  emailId: number;
}

export function MobileEmailDetailPage({ emailId }: MobileEmailDetailPageProps) {
  const { emails, selectEmail, removeEmail, patchEmail, updateEmail, upsertEmail } =
    useMailboxStore();

  const [isLoading, setIsLoading] = useState(true);
  const [emailDetail, setEmailDetail] = useState<Email | null>(null);
  const [currentTranslationLang, setCurrentTranslationLang] = useState<LanguageCode>();
  const [isTranslating, setIsTranslating] = useState(false);
  const { navigateToCompose, goBack } = useMobileNavigation();
  const getErrorMessage = (error: unknown, fallback: string) =>
    error instanceof Error && error.message ? error.message : fallback;

  // 加载邮件详情
  useEffect(() => {
    const loadEmailDetail = async () => {
      setIsLoading(true);
      try {
        // 先尝试从本地状态获取
        const localEmail = emails.find((email) => email.id === emailId);
        if (localEmail) {
          setEmailDetail(localEmail);
          selectEmail(localEmail);
        }

        // 如果本地没有找到，从API获取
        if (!localEmail) {
          const response = await apiClient.getEmailDetail(emailId);
          if (response.success && response.data) {
            setEmailDetail(response.data);
            upsertEmail(response.data);
            selectEmail(response.data);
          }
        }
      } catch (error) {
        console.error('Failed to load email detail:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadEmailDetail();
  }, [emailId, emails, selectEmail, upsertEmail]);

  useEffect(() => {
    const storeEmail = emails.find((email) => email.id === emailId);
    if (storeEmail) {
      setEmailDetail((prev) => (prev ? { ...prev, ...storeEmail } : storeEmail));
    }
  }, [emailId, emails]);

  // 处理回复
  const handleReply = () => {
    if (!emailDetail) return;
    navigateToCompose({ reply: emailDetail.id.toString() });
  };

  // 处理回复全部
  const handleReplyAll = () => {
    if (!emailDetail) return;
    navigateToCompose({ replyAll: emailDetail.id.toString() });
  };

  // 处理转发
  const handleForward = () => {
    if (!emailDetail) return;
    navigateToCompose({ forward: emailDetail.id.toString() });
  };

  // 处理星标
  const handleStar = async () => {
    if (!emailDetail) return;

    try {
      await apiClient.toggleEmailStar(emailDetail.id);
      const newStarStatus = !emailDetail.is_starred;
      setEmailDetail((prev) => (prev ? { ...prev, is_starred: newStarStatus } : null));
      updateEmail(emailDetail.id, { is_starred: newStarStatus });
      toast.success(newStarStatus ? '已添加星标' : '已移除星标');
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, '操作失败'));
    }
  };

  // 处理归档
  const handleArchive = async () => {
    if (!emailDetail) return;

    try {
      await apiClient.archiveEmail(emailDetail.id);
      removeEmail(emailDetail.id);
      toast.success('邮件已归档');
      goBack();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, '归档失败'));
    }
  };

  // 处理删除
  const handleDelete = async () => {
    if (!emailDetail) return;

    if (confirm('确定要删除这封邮件吗？')) {
      try {
        await apiClient.deleteEmail(emailDetail.id);
        removeEmail(emailDetail.id);
        toast.success('邮件已删除');
        goBack();
      } catch (error: unknown) {
        toast.error(getErrorMessage(error, '删除失败'));
      }
    }
  };

  // 处理标记已读/未读
  const handleToggleRead = async () => {
    if (!emailDetail) return;

    try {
      if (emailDetail.is_read) {
        await apiClient.markEmailAsUnread(emailDetail.id);
      } else {
        await apiClient.markEmailAsRead(emailDetail.id);
      }

      const newReadStatus = !emailDetail.is_read;
      setEmailDetail((prev) => (prev ? { ...prev, is_read: newReadStatus } : null));
      patchEmail(emailDetail.id, { is_read: newReadStatus });
      toast.success(newReadStatus ? '已标记为已读' : '已标记为未读');
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, '操作失败'));
    }
  };

  // 处理翻译
  const handleTranslate = (targetLang: LanguageCode) => {
    setIsTranslating(true);
    setCurrentTranslationLang(targetLang);
  };

  // 翻译完成回调
  const handleTranslationComplete = () => {
    setIsTranslating(false);
  };

  if (isLoading) {
    return (
      <MobileLayout>
        <MobilePage>
          <EmailDetailHeader subject="加载中..." />
          <MobileContent>
            <MobileLoading message="加载邮件详情..." />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  if (!emailDetail) {
    return (
      <MobileLayout>
        <MobilePage>
          <EmailDetailHeader subject="邮件不存在" />
          <MobileContent>
            <MobileEmptyState title="邮件不存在" description="请返回选择有效的邮件" />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  return (
    <MobileLayout>
      <MobilePage>
        <EmailDetailHeader
          subject={emailDetail.subject || '(无主题)'}
          email={emailDetail}
          onReply={handleReply}
          onReplyAll={handleReplyAll}
          onForward={handleForward}
          onDelete={handleDelete}
          onArchive={handleArchive}
          onToggleStar={handleStar}
          onToggleRead={handleToggleRead}
          onTranslate={handleTranslate}
          isTranslating={isTranslating}
          currentTranslationLang={currentTranslationLang}
        />

        {/* 邮件详情内容 */}
        <MobileContent padding={false} className="flex-1">
          <div className="flex h-full flex-col">
            <MobileEmailMetadata email={emailDetail} />
            <div className="flex-1 min-h-0">
              <EmailDetail
                email={emailDetail}
                hideHeader={true}
                translationLang={currentTranslationLang}
                isTranslating={isTranslating}
                onTranslationComplete={handleTranslationComplete}
              />
            </div>
          </div>
        </MobileContent>

        {/* 底部操作栏 - 优化为4个最常用功能 */}
        <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-3">
          <div className="flex items-center justify-around max-w-md mx-auto">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleReply}
              className="flex flex-col items-center gap-1 p-3 min-w-0 flex-1"
            >
              <Reply className="w-5 h-5" />
              <span className="text-xs">回复</span>
            </Button>

            <Button
              variant="ghost"
              size="sm"
              onClick={handleForward}
              className="flex flex-col items-center gap-1 p-3 min-w-0 flex-1"
            >
              <Forward className="w-5 h-5" />
              <span className="text-xs">转发</span>
            </Button>

            <Button
              variant="ghost"
              size="sm"
              onClick={handleStar}
              className="flex flex-col items-center gap-1 p-3 min-w-0 flex-1"
            >
              <Star
                className={`w-5 h-5 ${emailDetail.is_starred ? 'text-yellow-500 fill-current' : ''}`}
              />
              <span className="text-xs">星标</span>
            </Button>

            <Button
              variant="ghost"
              size="sm"
              onClick={handleDelete}
              className="flex flex-col items-center gap-1 p-3 min-w-0 flex-1 text-red-600 dark:text-red-400"
            >
              <Trash2 className="w-5 h-5" />
              <span className="text-xs">删除</span>
            </Button>
          </div>
        </div>
      </MobilePage>
    </MobileLayout>
  );
}

function MobileEmailMetadata({ email }: { email: Email }) {
  const [showRecipients, setShowRecipients] = useState(false);

  const fromAddress = parseEmailAddress(email.from);
  const toAddresses = parseEmailAddresses(email.to || '');
  const ccAddresses = parseEmailAddresses(email.cc || '');

  const formatDateTime = (dateString: string) => {
    const date = new Date(dateString);
    if (Number.isNaN(date.getTime())) {
      return dateString;
    }

    return date.toLocaleString('zh-CN', {
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const formatAddressList = (addresses: EmailAddress[], limit?: number) => {
    if (addresses.length === 0) return '';

    const visibleAddresses = typeof limit === 'number' ? addresses.slice(0, limit) : addresses;
    const formatted = visibleAddresses.map((address) => formatEmailAddress(address)).join(', ');

    if (typeof limit === 'number' && addresses.length > limit) {
      return `${formatted} 等${addresses.length}人`;
    }

    return formatted;
  };

  const hasExtraRecipients = toAddresses.length > 1 || ccAddresses.length > 0;

  return (
    <div className="border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-3 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">发件人</div>
          <div className="text-sm font-medium text-gray-900 dark:text-gray-100 break-words">
            {fromAddress ? formatEmailAddress(fromAddress) : email.from}
          </div>
        </div>

        <div className="text-right flex-shrink-0">
          <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">时间</div>
          <div className="text-sm text-gray-700 dark:text-gray-300">
            {formatDateTime(email.date)}
          </div>
        </div>
      </div>

      {toAddresses.length > 0 && (
        <div className="rounded-lg bg-gray-50 dark:bg-gray-900/40 px-3 py-2">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">收件人</div>
              <div className="text-sm text-gray-700 dark:text-gray-300 break-words">
                {showRecipients
                  ? formatAddressList(toAddresses)
                  : formatAddressList(toAddresses, 1)}
              </div>
            </div>

            {hasExtraRecipients && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setShowRecipients((prev) => !prev)}
                className="h-8 px-2 text-xs text-gray-500"
              >
                {showRecipients ? (
                  <>
                    收起
                    <ChevronUp className="w-4 h-4 ml-1" />
                  </>
                ) : (
                  <>
                    展开
                    <ChevronDown className="w-4 h-4 ml-1" />
                  </>
                )}
              </Button>
            )}
          </div>

          {showRecipients && ccAddresses.length > 0 && (
            <div className="mt-2 border-t border-gray-200 dark:border-gray-700 pt-2">
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">抄送</div>
              <div className="text-sm text-gray-700 dark:text-gray-300 break-words">
                {formatAddressList(ccAddresses)}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
