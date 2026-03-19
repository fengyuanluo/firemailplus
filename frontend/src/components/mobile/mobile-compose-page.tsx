'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'sonner';
import { useMailboxStore, useComposeStore } from '@/lib/store';
import { MobileLayout, MobilePage, MobileContent, MobileLoading } from './mobile-layout';
import { ComposeHeader } from './mobile-header';
import { RecipientInput } from '@/components/compose/recipient-input';
import { RichTextEditor } from '@/components/editor/rich-text-editor';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api';

export function MobileComposePage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const replyId = searchParams.get('reply');
  const replyAllId = searchParams.get('replyAll');
  const forwardId = searchParams.get('forward');
  const initializationKey = `${replyId || ''}|${replyAllId || ''}|${forwardId || ''}`;

  const { emails, accounts, selectedEmail, upsertAccount, upsertEmail } = useMailboxStore();
  const {
    mode,
    originalEmailId,
    draft,
    updateDraft,
    updateContent,
    updateRecipients,
    updateAttachments,
    clearDraft,
    setSendStatus,
    initializeReply,
    initializeReplyAll,
    initializeForward,
    openCompose,
  } = useComposeStore();

  // UI状态
  const [showCc, setShowCc] = useState(false);
  const [showBcc, setShowBcc] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const initializedKeyRef = useRef<string | null>(null);
  const selectedAccount = accounts.find((account) => account.id === draft.accountId);

  // 初始化邮件内容（回复、转发等）
  useEffect(() => {
    if (initializedKeyRef.current === initializationKey) {
      return;
    }

    let cancelled = false;

    const initializeComposeState = async () => {
      setIsInitializing(true);

      try {
        if (replyId || replyAllId || forwardId) {
          const emailId = parseInt(replyId || replyAllId || forwardId || '0', 10);

          if (Number.isNaN(emailId) || emailId <= 0) {
            throw new Error('原邮件ID无效');
          }

          let originalEmail =
            emails.find((email) => email.id === emailId) ||
            (selectedEmail?.id === emailId ? selectedEmail : undefined);

          if (!originalEmail) {
            const response = await apiClient.getEmailDetail(emailId);
            if (response.success && response.data) {
              originalEmail = response.data;
              upsertEmail(response.data);
            }
          }

          if (!originalEmail) {
            throw new Error('找不到原邮件');
          }

          if (cancelled) return;

          if (!accounts.some((account) => account.id === originalEmail.account_id)) {
            const accountResponse = await apiClient.getEmailAccount(originalEmail.account_id);
            if (accountResponse.success && accountResponse.data) {
              upsertAccount(accountResponse.data);
            }
          }

          if (cancelled) return;

          if (replyId) {
            initializeReply(originalEmail);
          } else if (replyAllId) {
            initializeReplyAll(originalEmail);
          } else if (forwardId) {
            initializeForward(originalEmail);
          }
        } else {
          openCompose();
        }

        initializedKeyRef.current = initializationKey;
      } catch (error) {
        console.error('初始化移动写信页失败:', error);
        toast.error(error instanceof Error ? error.message : '初始化写信页面失败');
        router.replace('/mailbox/mobile');
      } finally {
        if (!cancelled) {
          setIsInitializing(false);
        }
      }
    };

    initializeComposeState();

    return () => {
      cancelled = true;
    };
  }, [
    emails,
    forwardId,
    selectedEmail,
    initializeReply,
    initializeReplyAll,
    initializeForward,
    initializationKey,
    openCompose,
    router,
    replyAllId,
    replyId,
    upsertAccount,
    upsertEmail,
  ]);

  // 显示CC/BCC的逻辑
  useEffect(() => {
    setShowCc(draft.cc.length > 0);
    setShowBcc(draft.bcc.length > 0);
  }, [draft.cc.length, draft.bcc.length]);

  // 处理保存草稿
  const handleSave = async () => {
    try {
      // 构建邮件数据
      const emailData = {
        accountId: draft.accountId,
        to: draft.to.map((r) => r.email),
        cc: draft.cc.map((r) => r.email),
        bcc: draft.bcc.map((r) => r.email),
        subject: draft.subject,
        content: draft.content,
        htmlContent: draft.htmlContent,
        attachments: draft.attachments,
      };

      await apiClient.saveDraft(emailData);
      toast.success('草稿已保存');
    } catch (error) {
      console.error('保存草稿失败:', error);
      toast.error('保存草稿时发生错误');
    }
  };

  // 处理发送邮件
  const handleSend = async () => {
    // 验证必填字段
    if (!draft.to.length) {
      toast.warning('请至少添加一个收件人');
      return;
    }

    if (!draft.subject.trim()) {
      if (!confirm('邮件主题为空，确定要发送吗？')) {
        return;
      }
    }

    setIsSending(true);
    setSendStatus('sending');

    try {
      // 检查必要字段
      if (!draft.accountId) {
        throw new Error('请选择发件人账户');
      }

      // 提取已上传成功的附件ID
      const attachmentIds = draft.attachments
        .filter((attachment) => attachment.uploadStatus === 'completed' && attachment.attachmentId)
        .map((attachment) => attachment.attachmentId!);

      // 构建邮件数据
      const emailData = {
        account_id: draft.accountId,
        to: draft.to.map((r) => ({ address: r.email, name: r.name || r.email })),
        cc: draft.cc.map((r) => ({ address: r.email, name: r.name || r.email })),
        bcc: draft.bcc.map((r) => ({ address: r.email, name: r.name || r.email })),
        subject: draft.subject,
        text_body: draft.content,
        html_body: draft.htmlContent,
        attachment_ids: attachmentIds.length > 0 ? attachmentIds : undefined,
      };

      // 根据模式调用不同的API
      let response;
      switch (mode) {
        case 'reply':
          if (!originalEmailId) throw new Error('原邮件ID缺失');
          response = await apiClient.replyEmail(originalEmailId, emailData);
          break;
        case 'replyAll':
          if (!originalEmailId) throw new Error('原邮件ID缺失');
          response = await apiClient.replyAllEmail(originalEmailId, emailData);
          break;
        case 'forward':
          if (!originalEmailId) throw new Error('原邮件ID缺失');
          response = await apiClient.forwardEmail(originalEmailId, emailData);
          break;
        case 'compose':
        default:
          response = await apiClient.sendEmail(emailData);
          break;
      }

      if (!response.success) {
        throw new Error(response.message || '发送失败');
      }

      setSendStatus('sent');
      toast.success('邮件发送成功');
      clearDraft();
      router.back();
    } catch (error) {
      console.error('发送邮件失败:', error);
      setSendStatus('failed');
      toast.error('邮件发送失败，请稍后重试');
    } finally {
      setIsSending(false);
    }
  };

  // 处理丢弃邮件
  const handleDiscard = () => {
    const hasContent =
      draft.to.length > 0 ||
      draft.subject.trim() ||
      draft.content.trim() ||
      draft.htmlContent.trim() ||
      draft.attachments.length > 0;

    if (hasContent) {
      if (confirm('确定要丢弃这封邮件吗？未保存的内容将丢失。')) {
        clearDraft();
        router.back();
      }
    } else {
      clearDraft();
      router.back();
    }
  };

  if (isInitializing) {
    return (
      <MobileLayout>
        <MobilePage>
          <ComposeHeader />
          <MobileContent>
            <MobileLoading message="初始化写信页面..." />
          </MobileContent>
        </MobilePage>
      </MobileLayout>
    );
  }

  return (
    <MobileLayout>
      <MobilePage>
        <ComposeHeader
          onSave={handleSave}
          onSend={isSending ? undefined : handleSend}
          onDiscard={handleDiscard}
        />

        <MobileContent padding={false} className="flex-1">
          <div className="flex flex-col h-full">
            {/* 邮件头部信息 */}
            <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-4 space-y-4">
              {/* 发件人选择 */}
              <div>
                <Label className="text-sm text-gray-700 dark:text-gray-300 mb-2">发件人:</Label>
                <Select
                  value={draft.accountId?.toString() || ''}
                  onValueChange={(value) => updateDraft({ accountId: parseInt(value) })}
                >
                  <SelectTrigger className="w-full">
                    {selectedAccount ? (
                      <div className="flex flex-col items-start text-left">
                        <span className="font-medium">{selectedAccount.name || selectedAccount.email}</span>
                        <span className="text-xs text-gray-500">({selectedAccount.email})</span>
                      </div>
                    ) : (
                      <SelectValue placeholder="选择发件人账户" />
                    )}
                  </SelectTrigger>
                  <SelectContent>
                    {accounts.map((account) => (
                      <SelectItem key={account.id} value={account.id.toString()}>
                        <div className="flex flex-col items-start">
                          <span className="font-medium">{account.name || account.email}</span>
                          <span className="text-gray-500">({account.email})</span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* 收件人 */}
              <div>
                <RecipientInput
                  label="收件人"
                  placeholder="输入收件人邮箱地址"
                  recipients={draft.to}
                  onChange={(recipients) => updateRecipients('to', recipients)}
                  showContactPicker={true}
                />
              </div>

              {/* CC/BCC 切换按钮 */}
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowCc(!showCc)}
                  className="text-sm"
                >
                  {showCc ? '隐藏抄送' : '显示抄送'}
                </Button>

                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowBcc(!showBcc)}
                  className="text-sm"
                >
                  {showBcc ? '隐藏密送' : '显示密送'}
                </Button>
              </div>

              {/* 抄送 */}
              {showCc && (
                <div>
                  <RecipientInput
                    label="抄送"
                    placeholder="输入抄送邮箱地址"
                    recipients={draft.cc}
                    onChange={(recipients) => updateRecipients('cc', recipients)}
                  />
                </div>
              )}

              {/* 密送 */}
              {showBcc && (
                <div>
                  <RecipientInput
                    label="密送"
                    placeholder="输入密送邮箱地址"
                    recipients={draft.bcc}
                    onChange={(recipients) => updateRecipients('bcc', recipients)}
                  />
                </div>
              )}

              {/* 主题 */}
              <div>
                <Label className="text-sm text-gray-700 dark:text-gray-300 mb-2">主题:</Label>
                <Input
                  value={draft.subject}
                  onChange={(e) => updateDraft({ subject: e.target.value })}
                  placeholder="输入邮件主题"
                  className="w-full"
                />
              </div>
            </div>

            {/* 邮件正文编辑器（集成附件功能） */}
            <div className="flex-1 p-4">
              <RichTextEditor
                content={draft.htmlContent}
                placeholder="输入邮件内容..."
                onChange={(html, text) => updateContent(html, text)}
                minHeight="200px"
                maxHeight="none"
                className="text-sm"
                attachments={draft.attachments}
                onAttachmentsChange={updateAttachments}
                maxFileSize={25}
                maxFiles={10}
              />
            </div>
          </div>
        </MobileContent>
      </MobilePage>
    </MobileLayout>
  );
}
