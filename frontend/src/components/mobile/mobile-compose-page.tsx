'use client';

import { useEffect, useState, useCallback } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'sonner';
import { useMailboxStore, useComposeStore } from '@/lib/store';
import { validateEmailForm, showValidationErrors, autoSaveDraft, confirmClose } from '@/lib/compose-utils';
import { MobileLayout, MobilePage, MobileContent } from './mobile-layout';
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
import { ChevronDown, ChevronUp, Paperclip } from 'lucide-react';
import { apiClient } from '@/lib/api';

export function MobileComposePage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { emails, accounts, selectedEmail } = useMailboxStore();
  const {
    mode,
    originalEmailId,
    draft,
    updateDraft,
    updateContent,
    updateRecipients,
    updateAttachments,
    clearDraft,
    sendStatus,
    setSendStatus,
    autoSaveStatus,
    setAutoSaveStatus,
    initializeReply,
    initializeReplyAll,
    initializeForward,
    openCompose,
  } = useComposeStore();

  // UI状态
  const [showCc, setShowCc] = useState(false);
  const [showBcc, setShowBcc] = useState(false);
  const [isSending, setIsSending] = useState(false);

  // 自动保存草稿
  const handleAutoSave = useCallback(async () => {
    await autoSaveDraft(draft, setAutoSaveStatus);
  }, [draft, setAutoSaveStatus]);

  // 防抖自动保存
  useEffect(() => {
    const timer = setTimeout(() => {
      handleAutoSave();
    }, 2000);

    return () => clearTimeout(timer);
  }, [handleAutoSave]);

  // 初始化邮件内容（回复、转发等）
  useEffect(() => {
    const replyId = searchParams.get('reply');
    const replyAllId = searchParams.get('replyAll');
    const forwardId = searchParams.get('forward');

    if (replyId || replyAllId || forwardId) {
      const emailId = parseInt(replyId || replyAllId || forwardId || '0');
      // 优先从emails数组查找，如果找不到则从selectedEmail查找
      let originalEmail = emails.find((email) => email.id === emailId);

      // 如果emails数组中找不到，但selectedEmail的ID匹配，则使用selectedEmail
      if (!originalEmail && selectedEmail && selectedEmail.id === emailId) {
        originalEmail = selectedEmail;
      }

      if (originalEmail) {
        if (replyId) {
          initializeReply(originalEmail);
        } else if (replyAllId) {
          initializeReplyAll(originalEmail);
        } else if (forwardId) {
          initializeForward(originalEmail);
        }
      } else {
        toast.error('找不到原邮件');
        router.back();
      }
    } else {
      // 普通写信模式
      openCompose();
    }
  }, [
    searchParams,
    emails,
    selectedEmail,
    initializeReply,
    initializeReplyAll,
    initializeForward,
    openCompose,
    router,
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
    // 使用统一的表单验证
    const validation = validateEmailForm(draft);
    if (!validation.isValid) {
      showValidationErrors(validation.errors);
      return;
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
    confirmClose(draft, () => {
      clearDraft();
      router.back();
    });
  };

  return (
    <MobileLayout>
      <MobilePage className="mobile-compose-page">
        <div className="mobile-compose-header">
          <ComposeHeader
            onSave={handleSave}
            onSend={isSending ? undefined : handleSend}
            onDiscard={handleDiscard}
          />
        </div>

        {/* 自动保存状态提示 */}
        {autoSaveStatus && autoSaveStatus !== 'idle' && (
          <div className="mobile-autosave-status visible">
            {autoSaveStatus === 'saving' && '正在保存...'}
            {autoSaveStatus === 'saved' && '已保存'}
            {autoSaveStatus === 'error' && '保存失败'}
          </div>
        )}

        <MobileContent padding={false} className="mobile-compose-content">
          <div className="mobile-compose-form">
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
                    <SelectValue placeholder="选择发件人账户" />
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
                  className="mobile-input-field w-full"
                />
              </div>
            </div>

            {/* 邮件正文编辑器（集成附件功能） */}
            <div className="mobile-compose-editor flex-1 p-4">
              <RichTextEditor
                content={draft.htmlContent}
                placeholder="输入邮件内容..."
                onChange={(html, text) => updateContent(html, text)}
                minHeight="200px"
                maxHeight="none"
                className="text-sm mobile-editor-content"
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
