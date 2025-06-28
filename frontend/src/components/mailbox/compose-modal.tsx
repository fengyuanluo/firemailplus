'use client';

import { X, Save } from 'lucide-react';
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
import { useComposeStore, useMailboxStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { RichTextEditor } from '@/components/editor/rich-text-editor';
import { RecipientInput } from '@/components/compose/recipient-input';
import { TemplateSelector } from '@/components/compose/template-selector';
import { SendOptions } from '@/components/compose/send-options';
import { useEffect, useCallback, useState } from 'react';
import { toast } from 'sonner';

export function ComposeModal() {
  const {
    isOpen,
    setIsOpen,
    mode,
    originalEmailId,
    draft,
    updateDraft,
    updateContent,
    updateRecipients,
    updateAttachments,
    updateSendOptions,
    clearDraft,
    autoSaveStatus,
    sendStatus,
    setAutoSaveStatus,
    setSendStatus,
  } = useComposeStore();

  const { accounts, selectedAccount } = useMailboxStore();
  const [showCc, setShowCc] = useState(false);
  const [showBcc, setShowBcc] = useState(false);

  // 自动保存草稿
  const autoSaveDraft = useCallback(async () => {
    if (!draft.subject && !draft.content && !draft.htmlContent) return;

    setAutoSaveStatus('saving');
    try {
      // TODO: 调用API保存草稿
      await new Promise((resolve) => setTimeout(resolve, 1000)); // 模拟API调用
      setAutoSaveStatus('saved');
      setTimeout(() => setAutoSaveStatus('idle'), 2000);
    } catch (error) {
      setAutoSaveStatus('error');
      console.error('Auto save failed:', error);
    }
  }, [draft.subject, draft.content, draft.htmlContent, setAutoSaveStatus]);

  // 防抖自动保存
  useEffect(() => {
    if (!isOpen) return;

    const timer = setTimeout(() => {
      autoSaveDraft();
    }, 2000);

    return () => clearTimeout(timer);
  }, [autoSaveDraft, isOpen]);

  const handleClose = () => {
    if (draft.subject || draft.content || draft.htmlContent) {
      if (confirm('邮件内容尚未发送，确定要关闭吗？')) {
        setIsOpen(false);
      }
    } else {
      setIsOpen(false);
    }
  };

  // 处理模板选择
  const handleTemplateSelect = (template: any) => {
    updateContent(template.htmlBody, template.textBody);
    updateDraft({
      subject: template.subject,
      templateId: template.id,
    });
    toast.success(`已应用模板: ${template.name}`);
  };

  // 处理邮件预览
  const handlePreview = () => {
    // TODO: 实现邮件预览功能
    toast.info('邮件预览功能开发中...');
  };

  if (!isOpen) return null;

  // 处理定时发送
  const handleScheduleSend = (scheduledTime: string) => {
    updateSendOptions({
      ...draft.sendOptions,
      scheduledTime,
    });
    toast.success(`已设置定时发送: ${new Date(scheduledTime).toLocaleString()}`);
  };

  const handleSend = async () => {
    // 验证必填字段
    if (draft.to.length === 0) {
      toast.error('请输入收件人');
      return;
    }

    // 验证收件人邮箱格式
    const invalidRecipients = [...draft.to, ...draft.cc, ...draft.bcc].filter((r) => !r.isValid);
    if (invalidRecipients.length > 0) {
      toast.error('存在无效的邮箱地址，请检查后重试');
      return;
    }

    if (!draft.subject.trim()) {
      toast.error('请输入邮件主题');
      return;
    }

    if (!draft.content.trim() && !draft.htmlContent.trim()) {
      toast.error('请输入邮件内容');
      return;
    }

    // 检查是否有正在上传的附件
    const uploadingAttachments = draft.attachments.filter((a) => a.uploadStatus === 'uploading');
    if (uploadingAttachments.length > 0) {
      toast.error('请等待附件上传完成');
      return;
    }

    try {
      setSendStatus('sending');

      // 检查账户ID
      const accountId = draft.accountId || selectedAccount?.id;
      if (!accountId) {
        throw new Error('请选择发件账户');
      }

      // 提取已上传成功的附件ID
      const attachmentIds = draft.attachments
        .filter((attachment) => attachment.uploadStatus === 'completed' && attachment.attachmentId)
        .map((attachment) => attachment.attachmentId!);

      // 组装邮件数据
      const emailData = {
        account_id: accountId,
        to: draft.to.map((r) => ({ name: r.name || '', address: r.email })),
        cc: draft.cc.map((r) => ({ name: r.name || '', address: r.email })),
        bcc: draft.bcc.map((r) => ({ name: r.name || '', address: r.email })),
        subject: draft.subject,
        text_body: draft.content,
        html_body: draft.htmlContent,
        attachment_ids: attachmentIds.length > 0 ? attachmentIds : undefined,
        priority: draft.sendOptions.priority,
        importance: draft.sendOptions.importance,
        scheduled_time: draft.sendOptions.scheduledTime,
        request_read_receipt: draft.sendOptions.requestReadReceipt,
        request_delivery_receipt: draft.sendOptions.requestDeliveryReceipt,
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
      setIsOpen(false);
    } catch (error: any) {
      setSendStatus('failed');
      toast.error(`邮件发送失败: ${error.message}`);
      console.error('Send email failed:', error);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* 遮罩层 */}
      <div className="absolute inset-0 bg-black bg-opacity-50" onClick={handleClose} />

      {/* 弹窗内容 */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-5xl max-h-[90vh] flex flex-col">
        {/* 可滚动的内容区域 */}
        <div className="flex-1 overflow-auto">
          {/* 头部 - 可收缩 */}
          <div className="flex items-center justify-between px-4 py-1 border-b border-gray-200 dark:border-gray-700">
            <div className="flex items-center gap-3">
              <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {mode === 'reply'
                  ? '回复邮件'
                  : mode === 'replyAll'
                    ? '回复全部'
                    : mode === 'forward'
                      ? '转发邮件'
                      : '写信'}
              </h2>
              {/* 自动保存状态 */}
              {autoSaveStatus !== 'idle' && (
                <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                  {autoSaveStatus === 'saving' && (
                    <>
                      <div className="w-3 h-3 border border-gray-400 border-t-transparent rounded-full animate-spin"></div>
                      <span>保存中...</span>
                    </>
                  )}
                  {autoSaveStatus === 'saved' && (
                    <>
                      <Save className="w-3 h-3 text-green-500" />
                      <span>已保存</span>
                    </>
                  )}
                  {autoSaveStatus === 'error' && <span className="text-red-500">保存失败</span>}
                </div>
              )}
            </div>
            <Button variant="ghost" size="sm" onClick={handleClose} className="p-1 h-auto">
              <X className="w-4 h-4" />
            </Button>
          </div>

          {/* 邮件头部信息 - 可收缩 */}
          <div className="px-4 py-2 space-y-2 border-b border-gray-200 dark:border-gray-700">
            {/* 发件人选择 */}
            <div className="grid grid-cols-12 gap-3 items-center">
              <Label htmlFor="from" className="col-span-2 text-sm text-gray-700 dark:text-gray-300">
                发件人:
              </Label>
              <div className="col-span-10">
                <Select
                  value={draft.accountId?.toString() || selectedAccount?.id.toString()}
                  onValueChange={(value) => updateDraft({ accountId: parseInt(value) })}
                >
                  <SelectTrigger className="text-sm">
                    <SelectValue placeholder="选择发件账户" />
                  </SelectTrigger>
                  <SelectContent>
                    {accounts.map((account) => (
                      <SelectItem key={account.id} value={account.id.toString()}>
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{account.name}</span>
                          <span className="text-gray-500">({account.email})</span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* 收件人 */}
            <div className="grid grid-cols-12 gap-3 items-start">
              <div className="col-span-2 pt-2">
                <Label className="text-sm text-gray-700 dark:text-gray-300">收件人:</Label>
              </div>
              <div className="col-span-10">
                <RecipientInput
                  label=""
                  placeholder="输入收件人邮箱地址"
                  recipients={draft.to}
                  onChange={(recipients) => updateRecipients('to', recipients)}
                  showContactPicker={true}
                />

                {/* 显示/隐藏抄送密送按钮 */}
                <div className="flex gap-2 mt-2">
                  {!showCc && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowCc(true)}
                      className="text-xs text-blue-600 hover:text-blue-700 p-0 h-auto"
                    >
                      添加抄送
                    </Button>
                  )}
                  {!showBcc && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowBcc(true)}
                      className="text-xs text-blue-600 hover:text-blue-700 p-0 h-auto"
                    >
                      添加密送
                    </Button>
                  )}
                </div>
              </div>
            </div>

            {/* 抄送 */}
            {showCc && (
              <div className="grid grid-cols-12 gap-3 items-start">
                <div className="col-span-2 pt-2">
                  <Label className="text-sm text-gray-700 dark:text-gray-300">抄送:</Label>
                </div>
                <div className="col-span-10">
                  <RecipientInput
                    label=""
                    placeholder="输入抄送邮箱地址"
                    recipients={draft.cc}
                    onChange={(recipients) => updateRecipients('cc', recipients)}
                  />
                </div>
              </div>
            )}

            {/* 密送 */}
            {showBcc && (
              <div className="grid grid-cols-12 gap-3 items-start">
                <div className="col-span-2 pt-2">
                  <Label className="text-sm text-gray-700 dark:text-gray-300">密送:</Label>
                </div>
                <div className="col-span-10">
                  <RecipientInput
                    label=""
                    placeholder="输入密送邮箱地址"
                    recipients={draft.bcc}
                    onChange={(recipients) => updateRecipients('bcc', recipients)}
                  />
                </div>
              </div>
            )}

            {/* 主题和模板 */}
            <div className="grid grid-cols-12 gap-3 items-center">
              <Label
                htmlFor="subject"
                className="col-span-2 text-sm text-gray-700 dark:text-gray-300"
              >
                主题:
              </Label>
              <div className="col-span-8">
                <Input
                  id="subject"
                  type="text"
                  placeholder="输入邮件主题"
                  value={draft.subject}
                  onChange={(e) => updateDraft({ subject: e.target.value })}
                  className="text-sm"
                />
              </div>
              <div className="col-span-2">
                <TemplateSelector
                  onTemplateSelect={handleTemplateSelect}
                  selectedTemplateId={draft.templateId}
                />
              </div>
            </div>
          </div>

          {/* 邮件正文 - 自适应高度 */}
          <div className="flex-1 flex flex-col">
            {/* 富文本编辑器（集成附件功能） */}
            <div className="flex-1 px-4 py-2 flex flex-col">
              <RichTextEditor
                content={draft.htmlContent}
                placeholder="输入邮件内容..."
                onChange={updateContent}
                minHeight="200px"
                maxHeight="none"
                className="text-sm flex-1"
                attachments={draft.attachments}
                onAttachmentsChange={updateAttachments}
                maxFileSize={25}
                maxFiles={10}
              />
            </div>
          </div>
        </div>

        {/* 底部工具栏 - 固定 */}
        <div className="flex-shrink-0 flex items-center justify-between px-4 py-2 border-t border-gray-200 dark:border-gray-700">
          {/* 左侧：附件状态 */}
          <div className="flex items-center gap-2">
            {draft.attachments.length > 0 && (
              <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span>{draft.attachments.length} 个附件</span>
                {draft.attachments.some((a) => a.uploadStatus === 'uploading') && (
                  <span className="text-blue-600">上传中...</span>
                )}
                {draft.attachments.some((a) => a.uploadStatus === 'error') && (
                  <span className="text-red-600">上传失败</span>
                )}
              </div>
            )}
          </div>

          {/* 右侧：发送选项和操作按钮 */}
          <div className="flex items-center gap-2">
            <SendOptions
              options={draft.sendOptions}
              onChange={updateSendOptions}
              onSend={handleSend}
              onPreview={handlePreview}
              onSchedule={handleScheduleSend}
              isSending={sendStatus === 'sending'}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
