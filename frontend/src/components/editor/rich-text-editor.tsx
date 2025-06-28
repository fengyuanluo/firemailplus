'use client';

import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import TextStyle from '@tiptap/extension-text-style';
import Color from '@tiptap/extension-color';
import Link from '@tiptap/extension-link';
import Image from '@tiptap/extension-image';
import Table from '@tiptap/extension-table';
import TableRow from '@tiptap/extension-table-row';
import TableCell from '@tiptap/extension-table-cell';
import TableHeader from '@tiptap/extension-table-header';
import Highlight from '@tiptap/extension-highlight';
import Placeholder from '@tiptap/extension-placeholder';
import CharacterCount from '@tiptap/extension-character-count';
import Focus from '@tiptap/extension-focus';
import TextAlign from '@tiptap/extension-text-align';
import { EditorToolbar } from './editor-toolbar';
import { AttachmentUploadArea } from '@/components/compose/attachment-upload-area';
import { AttachmentBar } from '@/components/compose/attachment-bar';
import { validateFile, uploadFile } from '@/components/compose/attachment-upload-utils';
import { useAuthStore } from '@/lib/store';
import { useIsMobile } from '@/hooks/use-responsive';
import { toast } from 'sonner';
import { useEffect, useState, useRef } from 'react';

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

interface RichTextEditorProps {
  content?: string;
  placeholder?: string;
  onChange?: (html: string, text: string) => void;
  className?: string;
  editable?: boolean;
  minHeight?: string;
  maxHeight?: string;
  // 附件相关props
  attachments?: AttachmentFile[];
  onAttachmentsChange?: (
    attachments: AttachmentFile[] | ((prev: AttachmentFile[]) => AttachmentFile[])
  ) => void;
  maxFileSize?: number;
  maxFiles?: number;
  allowedTypes?: string[];
}

export function RichTextEditor({
  content = '',
  placeholder = '输入邮件内容...',
  onChange,
  className = '',
  editable = true,
  minHeight = '200px',
  maxHeight = '400px',
  attachments = [],
  onAttachmentsChange,
  maxFileSize = 25,
  maxFiles = 10,
  allowedTypes = [],
}: RichTextEditorProps) {
  const [showUploadArea, setShowUploadArea] = useState(false);
  const { token } = useAuthStore();
  const isMobile = useIsMobile();
  const mobileFileInputRef = useRef<HTMLInputElement>(null);
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        // 禁用默认的历史记录，使用自定义配置
        history: {
          depth: 50,
          newGroupDelay: 500,
        },
      }),
      TextStyle,
      Color,
      Link.configure({
        openOnClick: false,
        HTMLAttributes: {
          class: 'text-blue-600 hover:text-blue-800 underline',
          target: '_blank',
          rel: 'noopener noreferrer',
        },
      }),
      Image.configure({
        HTMLAttributes: {
          class: 'max-w-full h-auto rounded-lg',
        },
      }),
      Table.configure({
        resizable: true,
        HTMLAttributes: {
          class: 'border-collapse border border-gray-300 w-full',
        },
      }),
      TableRow.configure({
        HTMLAttributes: {
          class: 'border border-gray-300',
        },
      }),
      TableHeader.configure({
        HTMLAttributes: {
          class: 'border border-gray-300 bg-gray-100 dark:bg-gray-700 p-2 font-semibold',
        },
      }),
      TableCell.configure({
        HTMLAttributes: {
          class: 'border border-gray-300 p-2',
        },
      }),
      Highlight.configure({
        HTMLAttributes: {
          class: 'bg-yellow-200 dark:bg-yellow-800',
        },
      }),
      Placeholder.configure({
        placeholder,
        emptyEditorClass: 'is-editor-empty',
      }),
      CharacterCount,
      Focus.configure({
        className: 'has-focus',
        mode: 'all',
      }),
      TextAlign.configure({
        types: ['heading', 'paragraph'],
      }),
    ],
    content,
    editable,
    onUpdate: ({ editor }) => {
      const html = editor.getHTML();
      const text = editor.getText();
      onChange?.(html, text);
    },
    editorProps: {
      attributes: {
        class: `prose prose-sm max-w-none dark:prose-invert focus:outline-none ${className}`,
      },
    },
  });

  // 当外部content变化时更新编辑器内容
  useEffect(() => {
    if (editor && content !== editor.getHTML()) {
      editor.commands.setContent(content);
    }
  }, [content, editor]);

  // 更新附件状态
  const updateAttachment = (
    id: string,
    updates: Partial<AttachmentFile> | ((prev: AttachmentFile) => Partial<AttachmentFile>)
  ) => {
    if (!onAttachmentsChange) return;

    console.log('RichTextEditor: updateAttachment called for id', id, 'with updates', updates);

    // 使用函数式更新来确保获取最新的状态
    onAttachmentsChange((currentAttachments) => {
      const updatedAttachments = currentAttachments.map((attachment) => {
        if (attachment.id === id) {
          const newUpdates = typeof updates === 'function' ? updates(attachment) : updates;
          const updated = { ...attachment, ...newUpdates };
          console.log('RichTextEditor: updating attachment', id, 'from', attachment, 'to', updated);
          return updated;
        }
        return attachment;
      });

      console.log('RichTextEditor: returning updated attachments', updatedAttachments.length);
      return updatedAttachments;
    });
  };

  // 处理附件上传
  const handleFilesSelected = (files: FileList | File[]) => {
    if (!onAttachmentsChange) return;

    const fileArray = Array.from(files);
    const newAttachments: AttachmentFile[] = [];

    console.log('RichTextEditor: handleFilesSelected called with', files.length, 'files');
    console.log('RichTextEditor: current attachments.length =', attachments.length);

    fileArray.forEach((file) => {
      // 验证文件
      const error = validateFile(file, maxFileSize, maxFiles, attachments.length, allowedTypes);
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
      console.log(
        'RichTextEditor: calling onAttachmentsChange with',
        updatedAttachments.length,
        'attachments'
      );

      onAttachmentsChange(updatedAttachments);

      // 开始上传
      newAttachments.forEach((attachment) => {
        uploadFile(attachment, token, updateAttachment);
      });
    }
  };

  // 处理附件删除
  const handleAttachmentRemove = (id: string) => {
    if (!onAttachmentsChange) return;
    onAttachmentsChange(attachments.filter((attachment) => attachment.id !== id));
  };

  // 处理附件重试
  const handleAttachmentRetry = (id: string) => {
    const attachment = attachments.find((a) => a.id === id);
    if (attachment) {
      uploadFile(attachment, token, updateAttachment);
    }
  };

  // 处理移动端文件选择
  const handleMobileFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files) {
      handleFilesSelected(files);
    }
    // 清空input值，允许重复选择同一文件
    e.target.value = '';
  };

  // 处理附件按钮点击
  const handleAttachmentClick = () => {
    if (isMobile) {
      // 移动端：直接唤起文件选择器
      mobileFileInputRef.current?.click();
    } else {
      // 桌面端：显示上传区域
      setShowUploadArea(!showUploadArea);
    }
  };

  if (!editor) {
    return (
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg">
        <div className="p-4 text-center text-gray-500 dark:text-gray-400">正在加载编辑器...</div>
      </div>
    );
  }

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden flex flex-col relative">
      {/* 工具栏 - 固定在顶部 */}
      {editable && (
        <div className="flex-shrink-0">
          <EditorToolbar editor={editor} onAttachmentClick={handleAttachmentClick} />
        </div>
      )}

      {/* 附件上传区域（仅桌面端显示） */}
      {editable && !isMobile && (
        <AttachmentUploadArea
          isVisible={showUploadArea}
          onClose={() => setShowUploadArea(false)}
          onFilesSelected={handleFilesSelected}
          maxFileSize={maxFileSize}
          maxFiles={maxFiles}
          allowedTypes={allowedTypes}
          currentFileCount={attachments.length}
        />
      )}

      {/* 附件显示条 - 在编辑器顶部显示 */}
      {editable && attachments.length > 0 && (
        <div className="flex-shrink-0">
          <AttachmentBar
            attachments={attachments}
            onRemove={handleAttachmentRemove}
            onRetry={handleAttachmentRetry}
          />
        </div>
      )}

      {/* 移动端隐藏的文件输入 */}
      {editable && isMobile && (
        <input
          ref={mobileFileInputRef}
          type="file"
          multiple
          onChange={handleMobileFileSelect}
          className="hidden"
          accept={allowedTypes.length > 0 ? allowedTypes.join(',') : undefined}
        />
      )}

      {/* 编辑器内容 - 可滚动 */}
      <div
        className="relative flex-1"
        style={{
          minHeight: minHeight,
          maxHeight: maxHeight === 'none' ? undefined : maxHeight,
          overflowY: maxHeight === 'none' ? 'visible' : 'auto',
        }}
      >
        <EditorContent
          editor={editor}
          className="p-4 focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-opacity-50"
        />

        {/* 字符统计 */}
        {editable &&
          editor.extensionManager.extensions.find((ext) => ext.name === 'characterCount') && (
            <div className="absolute bottom-2 right-2 text-xs text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800 px-2 py-1 rounded shadow">
              {editor.storage.characterCount.characters()} 字符
            </div>
          )}
      </div>

      {/* 调试信息 */}
      {process.env.NODE_ENV === 'development' && (
        <div className="text-xs text-gray-500 p-2 border-t">
          调试: editable={editable.toString()}, attachments.length={attachments.length},
          attachments=
          {JSON.stringify(
            attachments.map((a) => ({ id: a.id, name: a.name, status: a.uploadStatus }))
          )}
        </div>
      )}
    </div>
  );
}

// 自定义样式
export const editorStyles = `
  .ProseMirror {
    outline: none;
    padding: 1rem;
    min-height: 200px;
  }

  .ProseMirror p.is-editor-empty:first-child::before {
    color: #adb5bd;
    content: attr(data-placeholder);
    float: left;
    height: 0;
    pointer-events: none;
  }

  .ProseMirror .has-focus {
    border-radius: 3px;
    box-shadow: 0 0 0 3px #68cef8;
  }

  .ProseMirror img {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin: 0.5rem 0;
  }

  .ProseMirror table {
    border-collapse: collapse;
    margin: 0;
    overflow: hidden;
    table-layout: fixed;
    width: 100%;
  }

  .ProseMirror table td,
  .ProseMirror table th {
    border: 1px solid #ced4da;
    box-sizing: border-box;
    min-width: 1em;
    padding: 3px 5px;
    position: relative;
    vertical-align: top;
  }

  .ProseMirror table th {
    background-color: #f8f9fa;
    font-weight: bold;
    text-align: left;
  }

  .ProseMirror table .selectedCell:after {
    background: rgba(200, 200, 255, 0.4);
    content: "";
    left: 0;
    right: 0;
    top: 0;
    bottom: 0;
    pointer-events: none;
    position: absolute;
    z-index: 2;
  }

  .ProseMirror blockquote {
    border-left: 4px solid #e9ecef;
    margin: 1.5rem 0;
    padding-left: 1rem;
    font-style: italic;
  }

  .ProseMirror code {
    background-color: #f8f9fa;
    border-radius: 4px;
    color: #e83e8c;
    font-size: 0.875em;
    padding: 0.25rem 0.375rem;
  }

  .ProseMirror pre {
    background: #f8f9fa;
    border-radius: 8px;
    color: #212529;
    font-family: 'JetBrainsMono', 'SFMono-Regular', 'Consolas', 'Liberation Mono', 'Menlo', monospace;
    margin: 1.5rem 0;
    padding: 0.75rem 1rem;
    white-space: pre-wrap;
  }

  .ProseMirror pre code {
    background: none;
    color: inherit;
    font-size: 0.8rem;
    padding: 0;
  }

  .ProseMirror ul,
  .ProseMirror ol {
    padding-left: 1rem;
  }

  .ProseMirror li p {
    margin-top: 0.25em;
    margin-bottom: 0.25em;
  }

  /* Dark mode styles */
  .dark .ProseMirror table th {
    background-color: #374151;
    color: #f9fafb;
  }

  .dark .ProseMirror table td,
  .dark .ProseMirror table th {
    border-color: #4b5563;
  }

  .dark .ProseMirror blockquote {
    border-left-color: #4b5563;
  }

  .dark .ProseMirror code {
    background-color: #374151;
    color: #f472b6;
  }

  .dark .ProseMirror pre {
    background: #374151;
    color: #f9fafb;
  }
`;
