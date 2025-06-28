'use client';

import { useState, useRef, useCallback } from 'react';
import { Upload, X, File, Image, FileText, Archive, Video, Music, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { formatFileSize } from '@/types/email';
import { API_BASE_URL } from '@/lib/api';
import { toast } from 'sonner';
import { useAuthStore } from '@/lib/store';

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

interface AttachmentManagerProps {
  attachments: AttachmentFile[];
  onChange: (attachments: AttachmentFile[]) => void;
  maxFileSize?: number; // MB
  maxFiles?: number;
  allowedTypes?: string[];
}

export function AttachmentManager({
  attachments,
  onChange,
  maxFileSize = 25,
  maxFiles = 10,
  allowedTypes = [],
}: AttachmentManagerProps) {
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { token } = useAuthStore();

  // 获取文件图标
  const getFileIcon = (type: string) => {
    const iconClass = 'w-5 h-5';

    if (type.startsWith('image/')) {
      return <Image className={`${iconClass} text-green-600`} />;
    } else if (type.startsWith('video/')) {
      return <Video className={`${iconClass} text-purple-600`} />;
    } else if (type.startsWith('audio/')) {
      return <Music className={`${iconClass} text-blue-600`} />;
    } else if (type.includes('pdf')) {
      return <FileText className={`${iconClass} text-red-600`} />;
    } else if (type.includes('zip') || type.includes('rar')) {
      return <Archive className={`${iconClass} text-orange-600`} />;
    } else {
      return <File className={`${iconClass} text-gray-600`} />;
    }
  };

  // 验证文件
  const validateFile = (file: File): string | null => {
    // 检查文件大小
    if (file.size > maxFileSize * 1024 * 1024) {
      return `文件大小不能超过 ${maxFileSize}MB`;
    }

    // 检查文件类型
    if (allowedTypes.length > 0 && !allowedTypes.includes(file.type)) {
      return '不支持的文件类型';
    }

    // 检查文件数量
    if (attachments.length >= maxFiles) {
      return `最多只能上传 ${maxFiles} 个文件`;
    }

    return null;
  };

  // 上传文件
  const uploadFile = async (attachmentFile: AttachmentFile) => {
    let progressInterval: NodeJS.Timeout | null = null;

    try {
      // 更新上传状态
      updateAttachment(attachmentFile.id, {
        uploadStatus: 'uploading',
        uploadProgress: 0,
      });

      // 创建FormData
      const formData = new FormData();
      formData.append('file', attachmentFile.file);

      // 改进的上传进度模拟
      const fileSize = attachmentFile.size;
      const baseDuration = Math.max(1000, Math.min((fileSize / 1024 / 1024) * 2000, 10000)); // 基于文件大小的持续时间，1-10秒
      const updateInterval = 100; // 更新间隔100ms
      const totalUpdates = baseDuration / updateInterval;
      let currentUpdate = 0;

      progressInterval = setInterval(() => {
        currentUpdate++;
        const baseProgress = currentUpdate / totalUpdates;

        // 使用非线性进度曲线：前70%较快，后30%较慢
        let adjustedProgress;
        if (baseProgress < 0.7) {
          adjustedProgress = baseProgress * 0.85; // 前70%映射到85%
        } else {
          adjustedProgress = 0.85 + (baseProgress - 0.7) * 0.05; // 后30%映射到5%
        }

        // 添加轻微随机波动，使进度看起来更真实
        const randomFactor = 0.95 + Math.random() * 0.1; // 0.95-1.05的随机因子
        const finalProgress = Math.min(adjustedProgress * randomFactor * 100, 90);

        updateAttachment(attachmentFile.id, (prev) => ({
          uploadProgress: Math.max(prev.uploadProgress, finalProgress),
        }));

        // 如果达到90%或超过预期时间，停止模拟
        if (finalProgress >= 90 || currentUpdate >= totalUpdates) {
          if (progressInterval) {
            clearInterval(progressInterval);
          }
        }
      }, updateInterval);

      // 调用上传API，添加超时控制
      const authToken = token || localStorage.getItem('auth_token');
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 60000); // 60秒超时

      try {
        const response = await fetch(`${API_BASE_URL}/attachments/upload`, {
          method: 'POST',
          body: formData,
          headers: {
            Authorization: `Bearer ${authToken}`,
          },
          signal: controller.signal,
        });

        clearTimeout(timeoutId);
        if (progressInterval) {
          clearInterval(progressInterval);
        }

        if (!response.ok) {
          // 尝试解析错误响应
          let errorMessage = '上传失败';
          try {
            const errorData = await response.json();
            if (errorData.message) {
              errorMessage = errorData.message;
            } else if (errorData.error) {
              errorMessage = errorData.error;
            }
          } catch (parseError) {
            // 如果无法解析JSON，根据状态码提供友好的错误信息
            switch (response.status) {
              case 400:
                errorMessage = '文件格式不正确或文件名无效';
                break;
              case 401:
                errorMessage = '登录已过期，请重新登录';
                break;
              case 403:
                errorMessage = '没有上传权限';
                break;
              case 413:
                errorMessage = '文件过大，最大支持25MB';
                break;
              case 415:
                errorMessage = '不支持的文件类型';
                break;
              case 500:
                errorMessage = '服务器内部错误，请稍后重试';
                break;
              case 503:
                errorMessage = '服务暂时不可用，请稍后重试';
                break;
              default:
                errorMessage = `上传失败 (错误代码: ${response.status})`;
            }
          }
          throw new Error(errorMessage);
        }

        const result = await response.json();

        // 更新完成状态
        updateAttachment(attachmentFile.id, {
          uploadStatus: 'completed',
          uploadProgress: 100,
          attachmentId: result.data.attachment_id,
        });

        toast.success(`${attachmentFile.name} 上传成功`);
      } catch (fetchError: any) {
        // 处理fetch相关的错误（网络错误、超时等）
        clearTimeout(timeoutId);
        if (progressInterval) {
          clearInterval(progressInterval);
        }
        throw fetchError;
      }
    } catch (error: any) {
      // 清理资源
      if (progressInterval) {
        clearInterval(progressInterval);
      }

      // 分析错误类型并提供友好的错误信息
      let errorMessage = '上传失败';
      let errorType = 'unknown';

      if (error.name === 'TypeError' && error.message.includes('fetch')) {
        // 网络连接错误
        errorMessage = '网络连接失败，请检查网络连接后重试';
        errorType = 'network';
      } else if (error.name === 'AbortError') {
        // 请求被取消
        errorMessage = '上传被取消';
        errorType = 'cancelled';
      } else if (error.message.includes('timeout')) {
        // 超时错误
        errorMessage = '上传超时，请检查网络连接或稍后重试';
        errorType = 'timeout';
      } else if (error.message) {
        // 使用服务器返回的错误信息
        errorMessage = error.message;
        errorType = 'server';
      }

      updateAttachment(attachmentFile.id, {
        uploadStatus: 'error',
        uploadProgress: 0,
        errorMessage: errorMessage,
      });

      // 根据错误类型显示不同的toast消息
      if (errorType === 'network') {
        toast.error(`${attachmentFile.name} 上传失败: ${errorMessage}`, {
          action: {
            label: '重试',
            onClick: () => uploadFile(attachmentFile),
          },
        });
      } else {
        toast.error(`${attachmentFile.name} 上传失败: ${errorMessage}`);
      }
    }
  };

  // 更新附件状态
  const updateAttachment = (
    id: string,
    updates: Partial<AttachmentFile> | ((prev: AttachmentFile) => Partial<AttachmentFile>)
  ) => {
    onChange(
      attachments.map((attachment) => {
        if (attachment.id === id) {
          const newUpdates = typeof updates === 'function' ? updates(attachment) : updates;
          return { ...attachment, ...newUpdates };
        }
        return attachment;
      })
    );
  };

  // 添加文件
  const addFiles = (files: FileList | File[]) => {
    const fileArray = Array.from(files);
    const newAttachments: AttachmentFile[] = [];

    fileArray.forEach((file) => {
      const error = validateFile(file);
      if (error) {
        toast.error(`${file.name}: ${error}`);
        return;
      }

      const attachmentFile: AttachmentFile = {
        id: `${Date.now()}-${Math.random()}`,
        file,
        name: file.name,
        size: file.size,
        type: file.type,
        uploadProgress: 0,
        uploadStatus: 'pending',
      };

      newAttachments.push(attachmentFile);
    });

    if (newAttachments.length > 0) {
      const updatedAttachments = [...attachments, ...newAttachments];
      onChange(updatedAttachments);

      // 开始上传
      newAttachments.forEach((attachment) => {
        uploadFile(attachment);
      });
    }
  };

  // 删除附件
  const removeAttachment = (id: string) => {
    onChange(attachments.filter((attachment) => attachment.id !== id));
  };

  // 重试上传
  const retryUpload = (id: string) => {
    const attachment = attachments.find((a) => a.id === id);
    if (attachment) {
      uploadFile(attachment);
    }
  };

  // 处理文件选择
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files) {
      addFiles(files);
    }
    // 清空input值，允许重复选择同一文件
    e.target.value = '';
  };

  // 处理拖拽
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragOver(false);

      const files = e.dataTransfer.files;
      if (files) {
        addFiles(files);
      }
    },
    [attachments, maxFiles, maxFileSize, allowedTypes]
  );

  return (
    <div className="space-y-4">
      {/* 上传区域 */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`
          border-2 border-dashed rounded-lg p-6 text-center transition-colors
          ${
            isDragOver
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
          }
        `}
      >
        <Upload className="w-8 h-8 text-gray-400 mx-auto mb-2" />
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
          拖拽文件到此处，或
          <Button
            variant="link"
            onClick={() => fileInputRef.current?.click()}
            className="p-0 h-auto text-blue-600 hover:text-blue-700"
          >
            点击选择文件
          </Button>
        </p>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          最大 {maxFileSize}MB，最多 {maxFiles} 个文件
        </p>

        <input
          ref={fileInputRef}
          type="file"
          multiple
          onChange={handleFileSelect}
          className="hidden"
          accept={allowedTypes.length > 0 ? allowedTypes.join(',') : undefined}
        />
      </div>

      {/* 附件列表 */}
      {attachments.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
            附件 ({attachments.length})
          </h4>

          <div className="space-y-2">
            {attachments.map((attachment) => (
              <div
                key={attachment.id}
                className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
              >
                {/* 文件图标 */}
                <div className="flex-shrink-0">{getFileIcon(attachment.type)}</div>

                {/* 文件信息 */}
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                    {attachment.name}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {formatFileSize(attachment.size)}
                  </div>

                  {/* 上传进度 */}
                  {attachment.uploadStatus === 'uploading' && (
                    <div className="mt-1">
                      <Progress value={attachment.uploadProgress} className="h-1" />
                    </div>
                  )}

                  {/* 错误信息 */}
                  {attachment.uploadStatus === 'error' && (
                    <div className="text-xs text-red-500 mt-1">{attachment.errorMessage}</div>
                  )}
                </div>

                {/* 状态和操作 */}
                <div className="flex items-center gap-2">
                  {attachment.uploadStatus === 'uploading' && (
                    <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                  )}

                  {attachment.uploadStatus === 'completed' && (
                    <div className="w-2 h-2 bg-green-500 rounded-full" />
                  )}

                  {attachment.uploadStatus === 'error' && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => retryUpload(attachment.id)}
                      className="text-xs text-red-600 hover:text-red-700"
                    >
                      重试
                    </Button>
                  )}

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeAttachment(attachment.id)}
                    className="p-1 h-auto text-gray-500 hover:text-red-600"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
