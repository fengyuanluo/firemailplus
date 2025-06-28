'use client';

import { useState, useRef, useEffect } from 'react';
import { Languages, ChevronDown, Loader2, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  SUPPORTED_LANGUAGES,
  LanguageCode,
  getUserPreferredLanguage,
  getLanguageName,
  openGoogleTranslate,
} from '@/lib/translate';

interface TranslateButtonProps {
  onTranslate: (targetLang: LanguageCode) => void;
  isTranslating?: boolean;
  currentLang?: LanguageCode;
  originalText?: string;
}

export function TranslateButton({
  onTranslate,
  isTranslating = false,
  currentLang,
  originalText = '',
}: TranslateButtonProps) {
  const [showLanguageMenu, setShowLanguageMenu] = useState(false);
  const [selectedLang, setSelectedLang] = useState<LanguageCode>(getUserPreferredLanguage());
  const menuRef = useRef<HTMLDivElement>(null);

  // 点击外部关闭菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setShowLanguageMenu(false);
      }
    };

    if (showLanguageMenu) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showLanguageMenu]);

  // 处理翻译
  const handleTranslate = (targetLang: LanguageCode) => {
    setSelectedLang(targetLang);
    setShowLanguageMenu(false);
    onTranslate(targetLang);
  };

  // 处理在新窗口打开Google翻译
  const handleOpenGoogleTranslate = () => {
    if (originalText) {
      openGoogleTranslate(originalText, selectedLang);
    }
    setShowLanguageMenu(false);
  };

  // 获取常用语言（前6个）
  const commonLanguages = SUPPORTED_LANGUAGES.slice(1, 7); // 排除'auto'
  const otherLanguages = SUPPORTED_LANGUAGES.slice(7);

  return (
    <div className="relative">
      {/* 翻译按钮 */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setShowLanguageMenu(!showLanguageMenu)}
        disabled={isTranslating}
        className="gap-1 h-8 px-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
        title="翻译邮件"
      >
        {isTranslating ? (
          <Loader2 className="w-4 h-4 animate-spin" />
        ) : (
          <Languages className="w-4 h-4" />
        )}
        <span className="text-xs">{isTranslating ? '翻译中...' : '翻译'}</span>
        {currentLang && currentLang !== 'auto' && (
          <span className="text-xs text-blue-600 dark:text-blue-400">
            ({getLanguageName(currentLang)})
          </span>
        )}
        <ChevronDown className="w-3 h-3" />
      </Button>

      {/* 语言选择菜单 */}
      {showLanguageMenu && (
        <div
          ref={menuRef}
          className="absolute top-full left-0 mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg z-20 min-w-[200px] max-h-[300px] overflow-y-auto"
        >
          <div className="py-1">
            {/* 常用语言 */}
            <div className="px-3 py-2 text-xs font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-600">
              常用语言
            </div>
            {commonLanguages.map((language) => (
              <button
                key={language.code}
                onClick={() => handleTranslate(language.code)}
                className={`
                  w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700
                  ${
                    selectedLang === language.code
                      ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                      : 'text-gray-700 dark:text-gray-300'
                  }
                `}
              >
                {language.name}
              </button>
            ))}

            {/* 分隔线 */}
            <div className="h-px bg-gray-200 dark:bg-gray-600 my-1" />

            {/* 其他语言 */}
            <div className="px-3 py-2 text-xs font-medium text-gray-500 dark:text-gray-400">
              其他语言
            </div>
            {otherLanguages.map((language) => (
              <button
                key={language.code}
                onClick={() => handleTranslate(language.code)}
                className={`
                  w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700
                  ${
                    selectedLang === language.code
                      ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                      : 'text-gray-700 dark:text-gray-300'
                  }
                `}
              >
                {language.name}
              </button>
            ))}

            {/* 分隔线 */}
            <div className="h-px bg-gray-200 dark:bg-gray-600 my-1" />

            {/* 在新窗口打开Google翻译 */}
            <button
              onClick={handleOpenGoogleTranslate}
              className="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 flex items-center gap-2"
            >
              <ExternalLink className="w-3 h-3" />
              在新窗口打开Google翻译
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
