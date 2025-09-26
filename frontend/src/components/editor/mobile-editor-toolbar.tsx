'use client';

import { Editor } from '@tiptap/react';
import {
  Bold,
  Italic,
  List,
  ListOrdered,
  Paperclip,
  MoreHorizontal,
  Undo,
  Redo,
  Link,
  Palette,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useState } from 'react';

interface MobileEditorToolbarProps {
  editor: Editor | null;
  onAttachmentClick?: () => void;
}

export function MobileEditorToolbar({ editor, onAttachmentClick }: MobileEditorToolbarProps) {
  const [showColorPicker, setShowColorPicker] = useState(false);

  if (!editor) {
    return null;
  }

  // 插入链接
  const insertLink = () => {
    const url = window.prompt('请输入链接地址:');
    if (url) {
      editor.chain().focus().setLink({ href: url }).run();
    }
  };

  // 设置文字颜色
  const setTextColor = (color: string) => {
    editor.chain().focus().setColor(color).run();
    setShowColorPicker(false);
  };

  // 常用颜色（简化版）
  const colors = [
    '#000000',
    '#666666',
    '#FF0000',
    '#0066FF',
    '#00AA00',
    '#FF6600',
  ];

  return (
    <div className="border-b border-gray-200 dark:border-gray-700 p-2 flex items-center gap-1 overflow-x-auto">
      {/* 撤销重做 */}
      <div className="flex items-center gap-1 flex-shrink-0">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().undo().run()}
          disabled={!editor.can().undo()}
          className="p-2 h-9 w-9 min-w-[36px]"
          title="撤销"
        >
          <Undo className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().redo().run()}
          disabled={!editor.can().redo()}
          className="p-2 h-9 w-9 min-w-[36px]"
          title="重做"
        >
          <Redo className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 基础格式化 */}
      <div className="flex items-center gap-1 flex-shrink-0">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleBold().run()}
          className={`p-2 h-9 w-9 min-w-[36px] ${
            editor.isActive('bold') ? 'bg-gray-200 dark:bg-gray-700' : ''
          }`}
          title="粗体"
        >
          <Bold className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleItalic().run()}
          className={`p-2 h-9 w-9 min-w-[36px] ${
            editor.isActive('italic') ? 'bg-gray-200 dark:bg-gray-700' : ''
          }`}
          title="斜体"
        >
          <Italic className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 列表 */}
      <div className="flex items-center gap-1 flex-shrink-0">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          className={`p-2 h-9 w-9 min-w-[36px] ${
            editor.isActive('bulletList') ? 'bg-gray-200 dark:bg-gray-700' : ''
          }`}
          title="无序列表"
        >
          <List className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          className={`p-2 h-9 w-9 min-w-[36px] ${
            editor.isActive('orderedList') ? 'bg-gray-200 dark:bg-gray-700' : ''
          }`}
          title="有序列表"
        >
          <ListOrdered className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 附件按钮 */}
      {onAttachmentClick && (
        <>
          <Button
            variant="ghost"
            size="sm"
            onClick={onAttachmentClick}
            className="p-2 h-9 w-9 min-w-[36px]"
            title="添加附件"
          >
            <Paperclip className="w-4 h-4" />
          </Button>
          <Separator orientation="vertical" className="h-6" />
        </>
      )}

      {/* 更多功能 */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="p-2 h-9 w-9 min-w-[36px]"
            title="更多功能"
          >
            <MoreHorizontal className="w-4 h-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem onClick={insertLink}>
            <Link className="w-4 h-4 mr-2" />
            插入链接
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setShowColorPicker(!showColorPicker)}>
            <Palette className="w-4 h-4 mr-2" />
            文字颜色
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => editor.chain().focus().toggleHighlight().run()}>
            高亮
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => editor.chain().focus().toggleStrike().run()}>
            删除线
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => editor.chain().focus().toggleBlockquote().run()}>
            引用
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* 颜色选择器 */}
      {showColorPicker && (
        <div className="absolute top-full left-0 right-0 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg p-2 z-10">
          <div className="grid grid-cols-6 gap-1">
            {colors.map((color) => (
              <button
                key={color}
                onClick={() => setTextColor(color)}
                className="w-8 h-8 rounded border border-gray-300 hover:scale-110 transition-transform"
                style={{ backgroundColor: color }}
                title={color}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
