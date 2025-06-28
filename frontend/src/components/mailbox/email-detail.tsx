'use client';

import { useState, useEffect } from 'react';
import { useMailboxStore } from '@/lib/store';
import { EmailHeader } from './email-header';
import { EmailContent } from './email-content';
import { AttachmentsFooter } from './attachments-footer';
import { LanguageCode } from '@/lib/translate';
import { apiClient } from '@/lib/api';
import type { Email } from '@/types/email';

interface EmailDetailProps {
  email?: Email | null; // 外部传入的邮件数据
  hideHeader?: boolean; // 是否隐藏头部，用于移动端
  translationLang?: LanguageCode; // 外部传入的翻译语言
  isTranslating?: boolean; // 外部传入的翻译状态
  onTranslationComplete?: (lang: LanguageCode) => void; // 外部传入的翻译完成回调
}

export function EmailDetail({
  email: externalEmail,
  hideHeader = false,
  translationLang: externalTranslationLang,
  isTranslating: externalIsTranslating,
  onTranslationComplete: externalOnTranslationComplete,
}: EmailDetailProps = {}) {
  const { selectedEmail: storeSelectedEmail } = useMailboxStore();

  // 优先使用外部传入的邮件，fallback到store中的邮件
  const selectedEmail = externalEmail || storeSelectedEmail;
  const [emailDetail, setEmailDetail] = useState(selectedEmail);
  const [isLoading, setIsLoading] = useState(false);
  const [currentTranslationLang, setCurrentTranslationLang] = useState<LanguageCode>();
  const [isTranslating, setIsTranslating] = useState(false);

  // 使用外部传入的翻译状态，如果没有则使用内部状态
  const finalTranslationLang = externalTranslationLang || currentTranslationLang;
  const finalIsTranslating =
    externalIsTranslating !== undefined ? externalIsTranslating : isTranslating;

  // 当选中邮件改变时，加载邮件详情
  useEffect(() => {
    if (selectedEmail) {
      loadEmailDetail(selectedEmail.id);
    } else {
      setEmailDetail(null);
    }
  }, [selectedEmail]);

  // 加载邮件详情
  const loadEmailDetail = async (emailId: number) => {
    setIsLoading(true);
    try {
      const response = await apiClient.getEmailDetail(emailId);
      if (response.success && response.data) {
        setEmailDetail(response.data);
      }
    } catch (error) {
      console.error('Failed to load email detail:', error);
      // 如果加载失败，使用列表中的邮件数据
      setEmailDetail(selectedEmail);
    } finally {
      setIsLoading(false);
    }
  };

  // 处理翻译
  const handleTranslate = (targetLang: LanguageCode) => {
    setIsTranslating(true);
    setCurrentTranslationLang(targetLang);
  };

  // 翻译完成回调
  const handleTranslationComplete = (lang: LanguageCode) => {
    setIsTranslating(false);
    // 如果有外部回调，调用外部回调
    if (externalOnTranslationComplete) {
      externalOnTranslationComplete(lang);
    }
  };

  // 如果没有选中邮件，显示空状态
  if (!selectedEmail && !emailDetail) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg
              className="w-8 h-8 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
            选择一封邮件
          </h3>
          <p className="text-gray-500 dark:text-gray-400">从左侧列表中选择一封邮件来查看详情</p>
        </div>
      </div>
    );
  }

  // 如果正在加载，显示加载状态
  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-900 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">正在加载邮件详情...</p>
        </div>
      </div>
    );
  }

  const email = emailDetail || selectedEmail;

  // 如果没有邮件数据，不渲染
  if (!email) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg
              className="w-8 h-8 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
            邮件加载失败
          </h3>
          <p className="text-gray-500 dark:text-gray-400">无法加载邮件详情，请重试</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* 邮件头部 - 固定不滚动 */}
      {!hideHeader && (
        <EmailHeader
          email={email}
          onTranslate={handleTranslate}
          isTranslating={finalIsTranslating}
          currentTranslationLang={finalTranslationLang}
        />
      )}

      {/* 邮件正文 - 可滚动 */}
      <EmailContent
        email={email}
        translationLang={finalTranslationLang}
        isTranslating={finalIsTranslating}
        onTranslationComplete={handleTranslationComplete}
      />

      {/* 附件列表 - 固定在底部 */}
      <AttachmentsFooter attachments={email.attachments || []} />
    </div>
  );
}
