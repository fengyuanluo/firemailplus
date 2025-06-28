'use client';

import { useState, useEffect } from 'react';
import DOMPurify from 'dompurify';
import { Email } from '@/types/email';
import { TranslationBar } from './translation-bar';
import { translateText, translateHtmlContent, LanguageCode, detectLanguage } from '@/lib/translate';
import { toast } from 'sonner';
import '@/styles/email-content.css';

interface EmailContentProps {
  email: Email;
  translationLang?: LanguageCode;
  isTranslating?: boolean;
  onTranslationComplete?: (lang: LanguageCode) => void;
}

export function EmailContent({
  email,
  translationLang,
  isTranslating = false,
  onTranslationComplete,
}: EmailContentProps) {
  const [translatedContent, setTranslatedContent] = useState<string>('');
  const [showTranslation, setShowTranslation] = useState(false);
  const [sourceLang, setSourceLang] = useState<LanguageCode>('auto');

  // 获取邮件内容
  const originalContent = email.html_body || email.text_body || '';
  const isHtmlContent = !!email.html_body;

  // 当翻译语言改变时，执行翻译
  useEffect(() => {
    if (translationLang && translationLang !== 'auto' && originalContent) {
      performTranslation();
    }
  }, [translationLang, originalContent]);

  // 执行翻译
  const performTranslation = async () => {
    if (!translationLang || !originalContent) return;

    try {
      // 检测源语言
      const detectedLang = detectLanguage(originalContent);
      setSourceLang(detectedLang);

      let translated: string;

      if (isHtmlContent) {
        // 翻译HTML内容
        translated = await translateHtmlContent(originalContent, translationLang, detectedLang);
      } else {
        // 翻译纯文本内容
        translated = await translateText(originalContent, translationLang, detectedLang);
      }

      setTranslatedContent(translated);
      setShowTranslation(true);
      onTranslationComplete?.(translationLang);

      toast.success('翻译完成');
    } catch (error: any) {
      console.error('Translation failed:', error);

      // 如果翻译失败，显示错误信息并提供Google翻译链接
      if (error.message.includes('请点击查看翻译结果')) {
        toast.error('翻译服务暂时不可用', {
          description: '点击翻译按钮中的"在新窗口打开Google翻译"查看翻译结果',
          duration: 5000,
        });
      } else {
        toast.error('翻译失败，请稍后重试');
      }

      onTranslationComplete?.(translationLang);
    }
  };

  // 显示原文
  const handleShowOriginal = () => {
    setShowTranslation(false);
  };

  // 关闭翻译
  const handleCloseTranslation = () => {
    setShowTranslation(false);
    setTranslatedContent('');
  };

  // 渲染邮件内容
  const renderContent = () => {
    const content = showTranslation && translatedContent ? translatedContent : originalContent;

    if (!content) {
      return (
        <div className="text-center py-8">
          <p className="text-gray-500 dark:text-gray-400">此邮件没有内容</p>
        </div>
      );
    }

    if (isHtmlContent) {
      // 使用DOMPurify清理HTML内容，确保安全
      const sanitizedHtml = DOMPurify.sanitize(content, {
        ALLOWED_TAGS: [
          // 基础文本标签
          'p',
          'br',
          'div',
          'span',
          'strong',
          'b',
          'em',
          'i',
          'u',
          's',
          'strike',
          'del',
          'ins',
          'sub',
          'sup',
          'small',
          'big',
          'mark',
          'abbr',
          'cite',
          'dfn',
          'kbd',
          'samp',
          'var',
          'time',
          // 标题标签
          'h1',
          'h2',
          'h3',
          'h4',
          'h5',
          'h6',
          // 列表标签
          'ul',
          'ol',
          'li',
          'dl',
          'dt',
          'dd',
          // 引用和代码
          'blockquote',
          'pre',
          'code',
          // 链接和媒体
          'a',
          'img',
          // 表格标签
          'table',
          'thead',
          'tbody',
          'tfoot',
          'tr',
          'th',
          'td',
          'caption',
          'colgroup',
          'col',
          // 分隔线
          'hr',
          // 邮件常见标签
          'font',
          'center',
          'address',
          'article',
          'section',
          'header',
          'footer',
          'main',
          'aside',
          'nav',
          'figure',
          'figcaption',
          // 表单相关（只读显示）
          'fieldset',
          'legend',
          'label',
          // 其他语义标签
          'details',
          'summary',
          'q',
          'ruby',
          'rt',
          'rp',
        ],
        ALLOWED_ATTR: [
          // 链接属性
          'href',
          'target',
          'rel',
          // 图片属性
          'src',
          'alt',
          'title',
          'width',
          'height',
          // 样式属性
          'style',
          'class',
          'id',
          // 表格属性
          'colspan',
          'rowspan',
          'cellpadding',
          'cellspacing',
          'border',
          'align',
          'valign',
          // 字体属性
          'color',
          'face',
          'size',
          // 通用属性
          'dir',
          'lang',
          'data-*',
          // 邮件特有属性
          'bgcolor',
          'background',
          'marginwidth',
          'marginheight',
          'leftmargin',
          'topmargin',
          'rightmargin',
          'bottommargin',
        ],
        ALLOWED_URI_REGEXP:
          /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|cid|xmpp|data):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i,
        // 允许更多样式属性
        ALLOW_DATA_ATTR: true,
        // 保留空白字符
        KEEP_CONTENT: true,
      });

      // 渲染HTML内容
      return (
        <div
          className="email-html-content prose prose-sm max-w-none dark:prose-invert prose-gray"
          dangerouslySetInnerHTML={{
            __html: sanitizedHtml,
          }}
          style={{
            // 确保HTML内容的样式适配
            wordBreak: 'break-word',
            lineHeight: '1.6',
            // 邮件特有样式
            fontSize: '14px',
            fontFamily: 'system-ui, -apple-system, sans-serif',
          }}
        />
      );
    } else {
      // 渲染纯文本内容
      return (
        <div className="whitespace-pre-wrap text-gray-700 dark:text-gray-300 leading-relaxed">
          {content}
        </div>
      );
    }
  };

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* 翻译状态栏 */}
      <TranslationBar
        isVisible={showTranslation && !!translatedContent}
        sourceLang={sourceLang}
        targetLang={translationLang || 'en'}
        originalText={originalContent}
        onShowOriginal={handleShowOriginal}
        onClose={handleCloseTranslation}
      />

      {/* 邮件正文 */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-6">
          {isTranslating ? (
            // 翻译加载状态
            <div className="space-y-4">
              <div className="animate-pulse">
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4 mb-2"></div>
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full mb-2"></div>
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-5/6 mb-2"></div>
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-2/3"></div>
              </div>
              <div className="text-center text-sm text-gray-500 dark:text-gray-400">
                正在翻译邮件内容...
              </div>
            </div>
          ) : (
            renderContent()
          )}
        </div>
      </div>
    </div>
  );
}
