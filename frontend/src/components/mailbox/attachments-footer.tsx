'use client';

import { Download, FileText, Image, Archive, Video, Music, File } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Attachment } from '@/types/email';
import { formatFileSize } from '@/types/email';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

interface AttachmentsFooterProps {
  attachments: Attachment[];
}

export function AttachmentsFooter({ attachments }: AttachmentsFooterProps) {
  if (!attachments || attachments.length === 0) {
    return null;
  }

  // 获取文件图标
  const getFileIcon = (contentType: string, filename: string) => {
    const iconClass = 'w-5 h-5';

    if (contentType.startsWith('image/')) {
      return <Image className={`${iconClass} text-green-600`} />;
    } else if (contentType.startsWith('video/')) {
      return <Video className={`${iconClass} text-purple-600`} />;
    } else if (contentType.startsWith('audio/')) {
      return <Music className={`${iconClass} text-blue-600`} />;
    } else if (contentType.includes('pdf')) {
      return <FileText className={`${iconClass} text-red-600`} />;
    } else if (
      contentType.includes('zip') ||
      contentType.includes('rar') ||
      contentType.includes('7z')
    ) {
      return <Archive className={`${iconClass} text-orange-600`} />;
    } else if (
      contentType.includes('document') ||
      contentType.includes('word') ||
      filename.endsWith('.doc') ||
      filename.endsWith('.docx')
    ) {
      return <FileText className={`${iconClass} text-blue-600`} />;
    } else if (
      contentType.includes('spreadsheet') ||
      contentType.includes('excel') ||
      filename.endsWith('.xls') ||
      filename.endsWith('.xlsx')
    ) {
      return <FileText className={`${iconClass} text-green-600`} />;
    } else if (
      contentType.includes('presentation') ||
      contentType.includes('powerpoint') ||
      filename.endsWith('.ppt') ||
      filename.endsWith('.pptx')
    ) {
      return <FileText className={`${iconClass} text-orange-600`} />;
    } else {
      return <File className={`${iconClass} text-gray-600`} />;
    }
  };

  // 处理附件下载
  const handleDownload = async (attachment: Attachment) => {
    try {
      const blob = await apiClient.downloadAttachment(attachment.id);

      // 创建下载链接
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = attachment.filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      toast.success(`已下载 ${attachment.filename}`);
    } catch (error: any) {
      console.error('Download failed:', error);
      toast.error(`下载失败: ${error.message || '未知错误'}`);
    }
  };

  // 获取文件类型描述
  const getFileTypeDescription = (contentType: string, filename: string) => {
    if (contentType.startsWith('image/')) {
      return '图片';
    } else if (contentType.startsWith('video/')) {
      return '视频';
    } else if (contentType.startsWith('audio/')) {
      return '音频';
    } else if (contentType.includes('pdf')) {
      return 'PDF文档';
    } else if (contentType.includes('zip') || contentType.includes('rar')) {
      return '压缩文件';
    } else if (contentType.includes('document') || contentType.includes('word')) {
      return 'Word文档';
    } else if (contentType.includes('spreadsheet') || contentType.includes('excel')) {
      return 'Excel表格';
    } else if (contentType.includes('presentation') || contentType.includes('powerpoint')) {
      return 'PowerPoint演示文稿';
    } else {
      // 从文件扩展名推断
      const ext = filename.split('.').pop()?.toLowerCase();
      switch (ext) {
        case 'txt':
          return '文本文件';
        case 'json':
          return 'JSON文件';
        case 'xml':
          return 'XML文件';
        case 'csv':
          return 'CSV文件';
        default:
          return '文件';
      }
    }
  };

  return (
    <div className="flex-shrink-0 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-750">
      <div className="p-4">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">
          附件 ({attachments.length})
        </h3>

        <div className="space-y-2">
          {attachments.map((attachment) => (
            <div
              key={attachment.id}
              className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500 transition-colors"
            >
              {/* 左侧：文件信息 */}
              <div className="flex items-center gap-3 flex-1 min-w-0">
                {/* 文件图标 */}
                <div className="flex-shrink-0">
                  {getFileIcon(attachment.content_type, attachment.filename)}
                </div>

                {/* 文件详情 */}
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                    {attachment.filename}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400 flex items-center gap-2">
                    <span>
                      {getFileTypeDescription(attachment.content_type, attachment.filename)}
                    </span>
                    <span>•</span>
                    <span>{formatFileSize(attachment.size)}</span>
                  </div>
                </div>
              </div>

              {/* 右侧：下载按钮 */}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleDownload(attachment)}
                className="p-2 h-auto flex-shrink-0"
                title={`下载 ${attachment.filename}`}
              >
                <Download className="w-4 h-4" />
              </Button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
