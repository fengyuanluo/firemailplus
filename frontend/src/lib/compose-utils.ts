'use client';

import { toast } from 'sonner';

// 收件人接口
interface Recipient {
  email: string;
  name?: string;
  isValid: boolean;
}

// 附件接口
interface AttachmentFile {
  id: string;
  file: File;
  name: string;
  size: number;
  type: string;
  uploadProgress: number;
  uploadStatus: 'pending' | 'uploading' | 'completed' | 'error';
  attachmentId?: number;
  errorMessage?: string;
}

// 草稿接口
interface Draft {
  to: Recipient[];
  cc: Recipient[];
  bcc: Recipient[];
  subject: string;
  content: string;
  htmlContent: string;
  attachments: AttachmentFile[];
}

// 表单验证结果
interface ValidationResult {
  isValid: boolean;
  errors: string[];
}

/**
 * 验证邮件表单
 * @param draft 草稿内容
 * @returns 验证结果
 */
export function validateEmailForm(draft: Draft): ValidationResult {
  const errors: string[] = [];

  // 验证收件人
  if (draft.to.length === 0) {
    errors.push('请输入收件人');
  }

  // 验证收件人邮箱格式
  const invalidRecipients = [...draft.to, ...draft.cc, ...draft.bcc].filter((r) => !r.isValid);
  if (invalidRecipients.length > 0) {
    errors.push('存在无效的邮箱地址，请检查后重试');
  }

  // 验证主题
  if (!draft.subject.trim()) {
    errors.push('请输入邮件主题');
  }

  // 验证内容
  if (!draft.content.trim() && !draft.htmlContent.trim()) {
    errors.push('请输入邮件内容');
  }

  // 检查是否有正在上传的附件
  const uploadingAttachments = draft.attachments.filter((a) => a.uploadStatus === 'uploading');
  if (uploadingAttachments.length > 0) {
    errors.push('请等待附件上传完成');
  }

  return {
    isValid: errors.length === 0,
    errors,
  };
}

/**
 * 显示验证错误
 * @param errors 错误列表
 */
export function showValidationErrors(errors: string[]): void {
  errors.forEach((error) => {
    toast.error(error);
  });
}

/**
 * 自动保存草稿
 * @param draft 草稿内容
 * @param setAutoSaveStatus 设置自动保存状态的函数
 */
export async function autoSaveDraft(
  draft: Draft,
  setAutoSaveStatus: (status: 'idle' | 'saving' | 'saved' | 'error') => void
): Promise<void> {
  // 如果没有内容，不保存
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
}

/**
 * 检查是否有未保存的内容
 * @param draft 草稿内容
 * @returns 是否有未保存的内容
 */
export function hasUnsavedContent(draft: Draft): boolean {
  return !!(draft.subject || draft.content || draft.htmlContent || draft.attachments.length > 0);
}

/**
 * 确认关闭对话框
 * @param draft 草稿内容
 * @param onConfirm 确认回调
 */
export function confirmClose(draft: Draft, onConfirm: () => void): void {
  if (hasUnsavedContent(draft)) {
    if (confirm('邮件内容尚未发送，确定要关闭吗？')) {
      onConfirm();
    }
  } else {
    onConfirm();
  }
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化后的文件大小
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/**
 * 生成唯一ID
 * @returns 唯一ID
 */
export function generateId(): string {
  return Math.random().toString(36).substr(2, 9);
}
