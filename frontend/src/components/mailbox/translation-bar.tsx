'use client';

import { X, RotateCcw, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LanguageCode, getLanguageName, openGoogleTranslate } from '@/lib/translate';

interface TranslationBarProps {
  isVisible: boolean;
  sourceLang?: LanguageCode;
  targetLang: LanguageCode;
  originalText?: string;
  onShowOriginal: () => void;
  onClose: () => void;
}

export function TranslationBar({
  isVisible,
  sourceLang,
  targetLang,
  originalText = '',
  onShowOriginal,
  onClose,
}: TranslationBarProps) {
  if (!isVisible) return null;

  const handleOpenGoogleTranslate = () => {
    if (originalText) {
      openGoogleTranslate(originalText, targetLang, sourceLang);
    }
  };

  return (
    <div className="flex items-center justify-between p-3 bg-blue-50 dark:bg-blue-900/20 border-b border-blue-200 dark:border-blue-800">
      {/* 左侧：翻译信息 */}
      <div className="flex items-center gap-2 text-sm">
        <span className="text-blue-700 dark:text-blue-300">
          已翻译为{getLanguageName(targetLang)}
        </span>
        {sourceLang && sourceLang !== 'auto' && (
          <span className="text-blue-600 dark:text-blue-400">
            (从{getLanguageName(sourceLang)})
          </span>
        )}
      </div>

      {/* 右侧：操作按钮 */}
      <div className="flex items-center gap-1">
        {/* 显示原文按钮 */}
        <Button
          variant="ghost"
          size="sm"
          onClick={onShowOriginal}
          className="h-7 px-2 text-blue-700 hover:text-blue-800 dark:text-blue-300 dark:hover:text-blue-200"
        >
          <RotateCcw className="w-3 h-3 mr-1" />
          <span className="text-xs">显示原文</span>
        </Button>

        {/* 在Google翻译中查看按钮 */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleOpenGoogleTranslate}
          className="h-7 px-2 text-blue-700 hover:text-blue-800 dark:text-blue-300 dark:hover:text-blue-200"
        >
          <ExternalLink className="w-3 h-3 mr-1" />
          <span className="text-xs">Google翻译</span>
        </Button>

        {/* 关闭按钮 */}
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="h-7 px-1 text-blue-700 hover:text-blue-800 dark:text-blue-300 dark:hover:text-blue-200"
        >
          <X className="w-3 h-3" />
        </Button>
      </div>
    </div>
  );
}
