'use client';

import { useState, useRef, useCallback } from 'react';
import { Upload, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useIsMobile } from '@/hooks/use-responsive';

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

interface AttachmentUploadAreaProps {
  onFilesSelected: (files: FileList | File[]) => void;
  maxFileSize?: number; // MB
  maxFiles?: number;
  allowedTypes?: string[];
  currentFileCount?: number;
  isVisible: boolean;
  onClose: () => void;
}

export function AttachmentUploadArea({
  onFilesSelected,
  maxFileSize = 25,
  maxFiles = 10,
  allowedTypes = [],
  currentFileCount = 0,
  isVisible,
  onClose,
}: AttachmentUploadAreaProps) {
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const isMobile = useIsMobile();

  // 处理文件选择
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files) {
      onFilesSelected(files);
    }
    // 清空input值，允许重复选择同一文件
    e.target.value = '';
    onClose();
  };

  // 处理拖拽（仅桌面端）
  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      if (isMobile) return;
      e.preventDefault();
      setIsDragOver(true);
    },
    [isMobile]
  );

  const handleDragLeave = useCallback(
    (e: React.DragEvent) => {
      if (isMobile) return;
      e.preventDefault();
      setIsDragOver(false);
    },
    [isMobile]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      if (isMobile) return;
      e.preventDefault();
      setIsDragOver(false);

      const files = e.dataTransfer.files;
      if (files) {
        onFilesSelected(files);
      }
      onClose();
    },
    [isMobile, onFilesSelected, onClose]
  );

  if (!isVisible) return null;

  return (
    <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">添加附件</h3>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="p-1 h-auto text-gray-500 hover:text-gray-700"
        >
          <X className="w-4 h-4" />
        </Button>
      </div>

      {/* 桌面端：显示拖拽区域 */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`
            border-2 border-dashed rounded-lg p-4 text-center transition-colors cursor-pointer
            ${
              isDragOver
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
            }
          `}
        onClick={() => fileInputRef.current?.click()}
      >
        <Upload className="w-6 h-6 text-gray-400 mx-auto mb-2" />
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">
          拖拽文件到此处，或点击选择文件
        </p>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          最大 {maxFileSize}MB，最多 {maxFiles} 个文件
        </p>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        multiple
        onChange={handleFileSelect}
        className="hidden"
        accept={allowedTypes.length > 0 ? allowedTypes.join(',') : undefined}
      />
    </div>
  );
}
