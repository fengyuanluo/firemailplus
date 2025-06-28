'use client';

import { Editor } from '@tiptap/react';
import {
  Bold,
  Italic,
  Underline,
  Strikethrough,
  List,
  ListOrdered,
  Quote,
  Link,
  Image,
  Table,
  Palette,
  Type,
  AlignLeft,
  AlignCenter,
  AlignRight,
  Undo,
  Redo,
  MoreHorizontal,
  Paperclip,
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

interface EditorToolbarProps {
  editor: Editor | null;
  onAttachmentClick?: () => void;
}

export function EditorToolbar({ editor, onAttachmentClick }: EditorToolbarProps) {
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

  // 插入图片
  const insertImage = () => {
    const url = window.prompt('请输入图片地址:');
    if (url) {
      editor.chain().focus().setImage({ src: url }).run();
    }
  };

  // 插入表格
  const insertTable = () => {
    editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run();
  };

  // 设置文字颜色
  const setTextColor = (color: string) => {
    editor.chain().focus().setColor(color).run();
    setShowColorPicker(false);
  };

  // 常用颜色
  const colors = [
    '#000000',
    '#333333',
    '#666666',
    '#999999',
    '#CCCCCC',
    '#FF0000',
    '#FF6600',
    '#FFCC00',
    '#00FF00',
    '#0066FF',
    '#6600FF',
    '#FF0066',
    '#FF3366',
    '#66FF33',
    '#3366FF',
  ];

  return (
    <div className="border-b border-gray-200 dark:border-gray-700 p-2 flex flex-wrap items-center gap-1">
      {/* 撤销重做 */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().undo().run()}
          disabled={!editor.can().undo()}
          className="p-2 h-8"
          title="撤销"
        >
          <Undo className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().redo().run()}
          disabled={!editor.can().redo()}
          className="p-2 h-8"
          title="重做"
        >
          <Redo className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 文本格式化 */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleBold().run()}
          className={`p-2 h-8 ${editor.isActive('bold') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="粗体"
        >
          <Bold className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleItalic().run()}
          className={`p-2 h-8 ${editor.isActive('italic') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="斜体"
        >
          <Italic className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleStrike().run()}
          className={`p-2 h-8 ${editor.isActive('strike') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="删除线"
        >
          <Strikethrough className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 列表 */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          className={`p-2 h-8 ${editor.isActive('bulletList') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="无序列表"
        >
          <List className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          className={`p-2 h-8 ${editor.isActive('orderedList') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="有序列表"
        >
          <ListOrdered className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => editor.chain().focus().toggleBlockquote().run()}
          className={`p-2 h-8 ${editor.isActive('blockquote') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="引用"
        >
          <Quote className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 插入功能 */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={insertLink}
          className={`p-2 h-8 ${editor.isActive('link') ? 'bg-gray-200 dark:bg-gray-700' : ''}`}
          title="插入链接"
        >
          <Link className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={insertImage}
          className="p-2 h-8"
          title="插入图片"
        >
          <Image className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={insertTable}
          className="p-2 h-8"
          title="插入表格"
        >
          <Table className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={onAttachmentClick}
          className="p-2 h-8"
          title="添加附件"
        >
          <Paperclip className="w-4 h-4" />
        </Button>
      </div>

      <Separator orientation="vertical" className="h-6" />

      {/* 文字颜色 */}
      <DropdownMenu open={showColorPicker} onOpenChange={setShowColorPicker}>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="p-2 h-8" title="文字颜色">
            <Palette className="w-4 h-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-48 p-2">
          <div className="grid grid-cols-5 gap-1">
            {colors.map((color) => (
              <button
                key={color}
                onClick={() => setTextColor(color)}
                className="w-6 h-6 rounded border border-gray-300 hover:scale-110 transition-transform"
                style={{ backgroundColor: color }}
                title={color}
              />
            ))}
          </div>
          <div className="mt-2 pt-2 border-t">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => editor.chain().focus().unsetColor().run()}
              className="w-full text-xs"
            >
              清除颜色
            </Button>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* 更多功能 */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="p-2 h-8" title="更多功能">
            <MoreHorizontal className="w-4 h-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem onClick={() => editor.chain().focus().setTextAlign('left').run()}>
            <AlignLeft className="w-4 h-4 mr-2" />
            左对齐
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => editor.chain().focus().setTextAlign('center').run()}>
            <AlignCenter className="w-4 h-4 mr-2" />
            居中对齐
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => editor.chain().focus().setTextAlign('right').run()}>
            <AlignRight className="w-4 h-4 mr-2" />
            右对齐
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
