import { API_BASE_URL } from '@/lib/api';
import { toast } from 'sonner';

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

// 验证文件
export const validateFile = (
  file: File,
  maxFileSize: number,
  maxFiles: number,
  currentCount: number,
  allowedTypes: string[]
): string | null => {
  // 检查文件大小
  if (file.size > maxFileSize * 1024 * 1024) {
    return `文件大小不能超过 ${maxFileSize}MB`;
  }

  // 检查文件类型
  if (allowedTypes.length > 0 && !allowedTypes.includes(file.type)) {
    return '不支持的文件类型';
  }

  // 检查文件数量
  if (currentCount >= maxFiles) {
    return `最多只能上传 ${maxFiles} 个文件`;
  }

  return null;
};

// 上传文件
export const uploadFile = async (
  attachmentFile: AttachmentFile,
  token: string | null,
  updateAttachment: (
    id: string,
    updates: Partial<AttachmentFile> | ((prev: AttachmentFile) => Partial<AttachmentFile>)
  ) => void
) => {
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
      const updateData = {
        uploadStatus: 'completed' as const,
        uploadProgress: 100,
        attachmentId: result.data.attachment_id,
      };

      // 调试日志
      if (process.env.NODE_ENV === 'development') {
        console.log('Upload completed, updating attachment:', {
          fileId: attachmentFile.id,
          fileName: attachmentFile.name,
          updateData,
          serverResponse: result,
        });
      }

      updateAttachment(attachmentFile.id, updateData);

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
          onClick: () => uploadFile(attachmentFile, token, updateAttachment),
        },
      });
    } else {
      toast.error(`${attachmentFile.name} 上传失败: ${errorMessage}`);
    }
  }
};
