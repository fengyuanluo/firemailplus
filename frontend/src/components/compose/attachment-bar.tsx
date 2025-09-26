'use client';

import { X, File, Image, FileText, Archive, Video, Music, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { formatFileSize } from '@/types/email';

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

interface AttachmentBarProps {
  attachments: AttachmentFile[];
  onRemove: (id: string) => void;
  onRetry?: (id: string) => void;
}

// 获取文件图标
const getFileIcon = (type: string) => {
  if (type.startsWith('image/')) {
    return <Image className="w-4 h-4 text-blue-500" />;
  }
  if (type.startsWith('video/')) {
    return <Video className="w-4 h-4 text-purple-500" />;
  }
  if (type.startsWith('audio/')) {
    return <Music className="w-4 h-4 text-green-500" />;
  }
  if (
    type.includes('pdf') ||
    type.includes('document') ||
    type.includes('text') ||
    type.includes('word') ||
    type.includes('excel') ||
    type.includes('powerpoint')
  ) {
    return <FileText className="w-4 h-4 text-red-500" />;
  }
  if (type.includes('zip') || type.includes('rar') || type.includes('archive')) {
    return <Archive className="w-4 h-4 text-orange-500" />;
  }
  return <File className="w-4 h-4 text-gray-500" />;
};

export function AttachmentBar({ attachments, onRemove, onRetry }: AttachmentBarProps) {
  // 调试日志
  if (process.env.NODE_ENV === 'development') {
    console.log('AttachmentBar rendered with:', {
      attachmentsLength: attachments.length,
      attachments: attachments.map((a) => ({ id: a.id, name: a.name, status: a.uploadStatus })),
    });
    console.log(
      'AttachmentBar will render:',
      attachments.length > 0 ? 'YES' : 'NO (returning null)'
    );
  }

  if (attachments.length === 0) {
    if (process.env.NODE_ENV === 'development') {
      console.log('AttachmentBar: returning null because attachments.length === 0');
    }
    return null;
  }

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-800">
      <div className="flex flex-wrap gap-2">
        {attachments.map((attachment) => (
          <div
            key={attachment.id}
            className="flex items-center gap-2 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 text-gray-700 dark:text-gray-200 text-xs px-3 py-2 rounded-lg shadow-sm min-w-0 max-w-xs hover:shadow-md transition-shadow"
          >
            {/* 文件图标 */}
            <div className="flex-shrink-0 text-blue-500 dark:text-blue-400">{getFileIcon(attachment.type)}</div>

            {/* 文件信息 */}
            <div className="flex-1 min-w-0">
              <div className="truncate font-medium text-gray-900 dark:text-gray-100">{attachment.name}</div>
              <div className="text-gray-500 dark:text-gray-400 text-xs">{formatFileSize(attachment.size)}</div>
            </div>

            {/* 状态指示器 */}
            <div className="flex-shrink-0 flex items-center gap-1">
              {attachment.uploadStatus === 'uploading' && (
                <>
                  <Loader2 className="w-3 h-3 animate-spin text-blue-500" />
                  <span className="text-xs text-gray-600 dark:text-gray-300">{attachment.uploadProgress}%</span>
                </>
              )}

              {attachment.uploadStatus === 'completed' && (
                <div className="w-2 h-2 bg-green-500 rounded-full" />
              )}

              {attachment.uploadStatus === 'error' && onRetry && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onRetry(attachment.id)}
                  className="text-xs text-red-500 hover:text-red-600 p-0 h-auto"
                >
                  重试
                </Button>
              )}

              {/* 删除按钮 */}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onRemove(attachment.id)}
                className="p-0 h-auto text-gray-400 hover:text-red-500 ml-1"
              >
                <X className="w-3 h-3" />
              </Button>
            </div>
          </div>
        ))}
      </div>

      {/* 上传进度条（仅在有文件正在上传时显示） */}
      {attachments.some((a) => a.uploadStatus === 'uploading') && (
        <div className="mt-2">
          {attachments
            .filter((a) => a.uploadStatus === 'uploading')
            .map((attachment) => (
              <div key={`progress-${attachment.id}`} className="mb-1">
                <div className="flex justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
                  <span className="truncate">{attachment.name}</span>
                  <span>{attachment.uploadProgress}%</span>
                </div>
                <Progress value={attachment.uploadProgress} className="h-1" />
              </div>
            ))}
        </div>
      )}
    </div>
  );
}
