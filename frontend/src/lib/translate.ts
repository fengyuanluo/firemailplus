/**
 * 翻译功能工具函数
 * 基于Google Translate免费服务实现
 */

// 支持的语言列表
export const SUPPORTED_LANGUAGES = [
  { code: 'auto', name: '自动检测' },
  { code: 'zh-CN', name: '中文(简体)' },
  { code: 'zh-TW', name: '中文(繁体)' },
  { code: 'en', name: 'English' },
  { code: 'ja', name: '日本語' },
  { code: 'ko', name: '한국어' },
  { code: 'es', name: 'Español' },
  { code: 'fr', name: 'Français' },
  { code: 'de', name: 'Deutsch' },
  { code: 'it', name: 'Italiano' },
  { code: 'pt', name: 'Português' },
  { code: 'ru', name: 'Русский' },
  { code: 'ar', name: 'العربية' },
  { code: 'th', name: 'ไทย' },
  { code: 'vi', name: 'Tiếng Việt' },
  { code: 'nl', name: 'Nederlands' },
  { code: 'sv', name: 'Svenska' },
  { code: 'tr', name: 'Türkçe' },
  { code: 'pl', name: 'Polski' },
  { code: 'el', name: 'Ελληνικά' },
] as const;

export type LanguageCode = (typeof SUPPORTED_LANGUAGES)[number]['code'];

// 翻译缓存
const translationCache = new Map<string, string>();

// 生成缓存键
function getCacheKey(text: string, targetLang: string): string {
  return `${targetLang}:${text.substring(0, 100)}`;
}

// 检测文本语言
export function detectLanguage(text: string): LanguageCode {
  // 简单的语言检测逻辑
  const chineseRegex = /[\u4e00-\u9fff]/;
  const japaneseRegex = /[\u3040-\u309f\u30a0-\u30ff]/;
  const koreanRegex = /[\uac00-\ud7af]/;
  const arabicRegex = /[\u0600-\u06ff]/;
  const thaiRegex = /[\u0e00-\u0e7f]/;

  if (chineseRegex.test(text)) {
    return 'zh-CN';
  } else if (japaneseRegex.test(text)) {
    return 'ja';
  } else if (koreanRegex.test(text)) {
    return 'ko';
  } else if (arabicRegex.test(text)) {
    return 'ar';
  } else if (thaiRegex.test(text)) {
    return 'th';
  } else {
    return 'en';
  }
}

// 获取用户首选语言
export function getUserPreferredLanguage(): LanguageCode {
  const browserLang = navigator.language.toLowerCase();

  if (browserLang.startsWith('zh-cn') || browserLang.startsWith('zh-hans')) {
    return 'zh-CN';
  } else if (browserLang.startsWith('zh-tw') || browserLang.startsWith('zh-hant')) {
    return 'zh-TW';
  } else if (browserLang.startsWith('ja')) {
    return 'ja';
  } else if (browserLang.startsWith('ko')) {
    return 'ko';
  } else if (browserLang.startsWith('es')) {
    return 'es';
  } else if (browserLang.startsWith('fr')) {
    return 'fr';
  } else if (browserLang.startsWith('de')) {
    return 'de';
  } else if (browserLang.startsWith('it')) {
    return 'it';
  } else if (browserLang.startsWith('pt')) {
    return 'pt';
  } else if (browserLang.startsWith('ru')) {
    return 'ru';
  } else if (browserLang.startsWith('ar')) {
    return 'ar';
  } else if (browserLang.startsWith('th')) {
    return 'th';
  } else if (browserLang.startsWith('vi')) {
    return 'vi';
  } else if (browserLang.startsWith('nl')) {
    return 'nl';
  } else if (browserLang.startsWith('sv')) {
    return 'sv';
  } else if (browserLang.startsWith('tr')) {
    return 'tr';
  } else if (browserLang.startsWith('pl')) {
    return 'pl';
  } else if (browserLang.startsWith('el')) {
    return 'el';
  } else {
    return 'en';
  }
}

// 使用Google Translate翻译文本
export async function translateText(
  text: string,
  targetLang: LanguageCode,
  sourceLang: LanguageCode = 'auto'
): Promise<string> {
  if (!text.trim()) {
    return text;
  }

  // 如果目标语言是自动检测，返回原文
  if (targetLang === 'auto') {
    return text;
  }

  // 检查缓存
  const cacheKey = getCacheKey(text, targetLang);
  if (translationCache.has(cacheKey)) {
    return translationCache.get(cacheKey)!;
  }

  try {
    // 使用Google Translate免费服务
    const translateUrl = 'https://translate.googleapis.com/translate_a/single';
    const params = new URLSearchParams({
      client: 'gtx',
      sl: sourceLang,
      tl: targetLang,
      dt: 't',
      q: text,
    });

    const response = await fetch(`${translateUrl}?${params}`, {
      method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
    });

    if (!response.ok) {
      throw new Error(`Translation failed: ${response.status}`);
    }

    const data = await response.json();

    // 解析翻译结果
    let translatedText = '';
    if (data && data[0]) {
      for (const item of data[0]) {
        if (item[0]) {
          translatedText += item[0];
        }
      }
    }

    if (!translatedText) {
      throw new Error('No translation result');
    }

    // 缓存翻译结果
    translationCache.set(cacheKey, translatedText);

    return translatedText;
  } catch (error) {
    console.error('Translation error:', error);

    // 翻译失败时，尝试使用备用方案（在新窗口打开Google翻译）
    const fallbackUrl = `https://translate.google.com/translate?sl=${sourceLang}&tl=${targetLang}&text=${encodeURIComponent(text)}`;

    // 返回原文并提示用户
    throw new Error(`翻译失败，请点击查看翻译结果: ${fallbackUrl}`);
  }
}

// 在新窗口打开Google翻译
export function openGoogleTranslate(
  text: string,
  targetLang: LanguageCode,
  sourceLang: LanguageCode = 'auto'
): void {
  const translateUrl = `https://translate.google.com/translate?sl=${sourceLang}&tl=${targetLang}&text=${encodeURIComponent(text)}`;
  window.open(translateUrl, '_blank', 'noopener,noreferrer');
}

// 翻译HTML内容（保留HTML标签）
export async function translateHtmlContent(
  htmlContent: string,
  targetLang: LanguageCode,
  sourceLang: LanguageCode = 'auto'
): Promise<string> {
  if (!htmlContent.trim()) {
    return htmlContent;
  }

  try {
    // 创建临时DOM元素来处理HTML
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = htmlContent;

    // 收集所有文本节点
    const textNodes: { node: Text; originalText: string }[] = [];
    const walker = document.createTreeWalker(tempDiv, NodeFilter.SHOW_TEXT, {
      acceptNode: (node) => {
        const text = node.textContent?.trim();
        // 只处理有实际内容的文本节点
        return text && text.length > 0 ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT;
      },
    });

    let node;
    while ((node = walker.nextNode())) {
      const textNode = node as Text;
      const originalText = textNode.textContent || '';
      if (originalText.trim()) {
        textNodes.push({ node: textNode, originalText });
      }
    }

    // 如果没有文本节点，返回原HTML
    if (textNodes.length === 0) {
      return htmlContent;
    }

    // 批量翻译所有文本内容
    const textsToTranslate = textNodes.map((item) => item.originalText);
    const combinedText = textsToTranslate.join('\n\n---SEPARATOR---\n\n');

    const translatedCombined = await translateText(combinedText, targetLang, sourceLang);
    const translatedTexts = translatedCombined.split('\n\n---SEPARATOR---\n\n');

    // 确保翻译结果数量匹配
    if (translatedTexts.length !== textNodes.length) {
      // 如果分割结果不匹配，尝试逐个翻译
      for (let i = 0; i < textNodes.length; i++) {
        try {
          const translated = await translateText(textNodes[i].originalText, targetLang, sourceLang);
          textNodes[i].node.textContent = translated;
        } catch (error) {
          // 如果单个翻译失败，保持原文
          console.warn('Failed to translate text node:', textNodes[i].originalText, error);
        }
      }
    } else {
      // 替换文本节点内容
      for (let i = 0; i < textNodes.length; i++) {
        textNodes[i].node.textContent = translatedTexts[i] || textNodes[i].originalText;
      }
    }

    return tempDiv.innerHTML;
  } catch (error) {
    console.error('HTML translation error:', error);
    throw error;
  }
}

// 清理翻译缓存
export function clearTranslationCache(): void {
  translationCache.clear();
}

// 获取语言名称
export function getLanguageName(code: LanguageCode): string {
  const language = SUPPORTED_LANGUAGES.find((lang) => lang.code === code);
  return language?.name || code;
}
