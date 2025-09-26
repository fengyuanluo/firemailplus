'use client';

import { X, File, Image, FileText, Archive, Video, Music, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { formatFileSize } from '@/lib/compose-utils';

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

interface MobileAttachmentBarProps {
  attachments: AttachmentFile[];
  onRemove: (id: string) => void;
  onRetry?: (id: string) => void;
}

// 获取文件类型图标
function getFileIcon(type: string) {
  if (type.startsWith('image/')) return Image;
  if (type.startsWith('video/')) return Video;
  if (type.startsWith('audio/')) return Music;
  if (type.includes('pdf') || type.includes('document') || type.includes('text')) return FileText;
  if (type.includes('zip') || type.includes('rar') || type.includes('archive')) return Archive;
  return File;
}

// 获取文件名的简短版本
function getShortFileName(name: string, maxLength: number = 20): string {
  if (name.length <= maxLength) return name;
  
  const extension = name.split('.').pop() || '';
  const nameWithoutExt = name.substring(0, name.lastIndexOf('.'));
  const maxNameLength = maxLength - extension.length - 4; // 4 for "..." and "."
  
  if (maxNameLength <= 0) return name;
  
  return `${nameWithoutExt.substring(0, maxNameLength)}...${extension}`;
}

export function MobileAttachmentBar({ attachments, onRemove, onRetry }: MobileAttachmentBarProps) {
  if (attachments.length === 0) return null;

  return (
    <div className="mobile-attachment-bar border-t border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-800">
      <div className="mobile-attachment-list space-y-2">
        {attachments.map((attachment) => {
          const Icon = getFileIcon(attachment.type);
          const shortName = getShortFileName(attachment.name);
          const isUploading = attachment.uploadStatus === 'uploading';
          const hasError = attachment.uploadStatus === 'error';

          return (
            <div
              key={attachment.id}
              className={`mobile-attachment-item ${hasError ? 'border-red-300 bg-red-50' : ''}`}
            >
              {/* 文件图标 */}
              <div className="flex-shrink-0">
                {isUploading ? (
                  <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                ) : (
                  <Icon className={`w-4 h-4 ${hasError ? 'text-red-500' : 'text-gray-500'}`} />
                )}
              </div>

              {/* 文件信息 */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between">
                  <span
                    className={`text-sm font-medium truncate ${
                      hasError ? 'text-red-700' : 'text-gray-900'
                    }`}
                    title={attachment.name}
                  >
                    {shortName}
                  </span>
                  <span className="text-xs text-gray-500 ml-2 flex-shrink-0">
                    {formatFileSize(attachment.size)}
                  </span>
                </div>

                {/* 上传进度 */}
                {isUploading && (
                  <div className="mt-1">
                    <Progress value={attachment.uploadProgress} className="h-1" />
                    <span className="text-xs text-gray-500">
                      {attachment.uploadProgress}%
                    </span>
                  </div>
                )}

                {/* 错误信息 */}
                {hasError && attachment.errorMessage && (
                  <div className="text-xs text-red-600 mt-1 truncate">
                    {attachment.errorMessage}
                  </div>
                )}
              </div>

              {/* 操作按钮 */}
              <div className="flex-shrink-0 ml-2">
                {hasError && onRetry ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRetry(attachment.id)}
                    className="h-6 w-6 p-0 text-blue-600 hover:text-blue-700"
                    title="重试上传"
                  >
                    <Loader2 className="w-3 h-3" />
                  </Button>
                ) : (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRemove(attachment.id)}
                    className="h-6 w-6 p-0 text-gray-400 hover:text-red-600"
                    title="删除附件"
                    disabled={isUploading}
                  >
                    <X className="w-3 h-3" />
                  </Button>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* 附件统计 */}
      <div className="mt-2 text-xs text-gray-500 text-center">
        {attachments.length} 个附件 • 总大小:{' '}
        {formatFileSize(attachments.reduce((total, file) => total + file.size, 0))}
      </div>
    </div>
  );
}
