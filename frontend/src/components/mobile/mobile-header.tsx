'use client';

import {
  ArrowLeft,
  Search,
  MoreVertical,
  Edit,
  ReplyAll,
  Archive,
  CheckCheck,
  Star,
  Languages,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
} from '@/components/ui/dropdown-menu';
import { useMobileNavigation } from '@/hooks/use-mobile-navigation';
import type { Email } from '@/types/email';
import { SUPPORTED_LANGUAGES, LanguageCode, getLanguageName } from '@/lib/translate';

interface MobileHeaderProps {
  title: string;
  showBack?: boolean;
  showSearch?: boolean;
  showCompose?: boolean;
  showMore?: boolean;
  onBack?: () => void;
  onSearch?: () => void;
  onCompose?: () => void;
  onMore?: () => void;
  rightContent?: React.ReactNode;
}

export function MobileHeader({
  title,
  showBack = false,
  showSearch = false,
  showCompose = false,
  showMore = false,
  onBack,
  onSearch,
  onCompose,
  onMore,
  rightContent,
}: MobileHeaderProps) {
  const { navigateToSearch, navigateToCompose, goBack } = useMobileNavigation();

  const handleBack = () => {
    if (onBack) {
      onBack();
    } else {
      goBack();
    }
  };

  const handleSearch = () => {
    if (onSearch) {
      onSearch();
    } else {
      navigateToSearch();
    }
  };

  const handleCompose = () => {
    if (onCompose) {
      onCompose();
    } else {
      navigateToCompose();
    }
  };

  return (
    <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-3">
      <div className="flex items-start justify-between gap-3">
        {/* 左侧区域 */}
        <div className="flex items-start gap-3 flex-1 min-w-0">
          {showBack && (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleBack}
              className="p-2 h-auto flex-shrink-0 mt-0.5"
            >
              <ArrowLeft className="w-5 h-5" />
            </Button>
          )}

          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100 break-words leading-tight">
            {title}
          </h1>
        </div>

        {/* 右侧区域 */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {rightContent}

          {showSearch && (
            <Button variant="ghost" size="sm" onClick={handleSearch} className="p-2 h-auto">
              <Search className="w-5 h-5" />
            </Button>
          )}

          {showCompose && (
            <Button variant="ghost" size="sm" onClick={handleCompose} className="p-2 h-auto">
              <Edit className="w-5 h-5" />
            </Button>
          )}

          {showMore && (
            <Button variant="ghost" size="sm" onClick={onMore} className="p-2 h-auto">
              <MoreVertical className="w-5 h-5" />
            </Button>
          )}
        </div>
      </div>
    </header>
  );
}

// 简化版头部组件
interface SimpleHeaderProps {
  title: string;
  onBack?: () => void;
}

export function SimpleHeader({ title, onBack }: SimpleHeaderProps) {
  return <MobileHeader title={title} showBack={true} onBack={onBack} />;
}

// 邮箱列表头部
export function AccountsHeader() {
  return <MobileHeader title="邮箱" showSearch={true} showCompose={true} />;
}

// 文件夹列表头部
interface FoldersHeaderProps {
  accountName: string;
}

export function FoldersHeader({ accountName }: FoldersHeaderProps) {
  return <MobileHeader title={accountName} showBack={true} showSearch={true} showCompose={true} />;
}

// 邮件列表头部
interface EmailsHeaderProps {
  folderName: string;
}

export function EmailsHeader({ folderName }: EmailsHeaderProps) {
  return <MobileHeader title={folderName} showBack={true} showSearch={true} showCompose={true} />;
}

// 邮件详情头部
interface EmailDetailHeaderProps {
  subject: string;
  email?: Email; // 邮件数据，用于菜单操作
  onReply?: () => void;
  onReplyAll?: () => void;
  onForward?: () => void;
  onDelete?: () => void;
  onArchive?: () => void;
  onToggleStar?: () => void;
  onToggleRead?: () => void;
  onTranslate?: (targetLang: LanguageCode) => void;
  isTranslating?: boolean;
  currentTranslationLang?: LanguageCode;
}

export function EmailDetailHeader({
  subject,
  email,
  onReply,
  onReplyAll,
  onForward,
  onDelete,
  onArchive,
  onToggleStar,
  onToggleRead,
  onTranslate,
  isTranslating,
  currentTranslationLang,
}: EmailDetailHeaderProps) {
  // 获取常用语言（前6个，排除'auto'）
  const commonLanguages = SUPPORTED_LANGUAGES.slice(1, 7);
  return (
    <MobileHeader
      title={subject || '(无主题)'}
      showBack={true}
      rightContent={
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="p-2 h-auto">
              <MoreVertical className="w-5 h-5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            {onReplyAll && (
              <DropdownMenuItem onClick={onReplyAll}>
                <ReplyAll className="w-4 h-4 mr-2" />
                回复全部
              </DropdownMenuItem>
            )}

            <DropdownMenuSeparator />

            {/* 翻译子菜单 */}
            {onTranslate && (
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <Languages className="w-4 h-4 mr-2" />
                  {isTranslating ? '翻译中...' : '翻译'}
                  {currentTranslationLang && currentTranslationLang !== 'auto' && (
                    <span className="ml-1 text-xs text-blue-600 dark:text-blue-400">
                      ({getLanguageName(currentTranslationLang)})
                    </span>
                  )}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent className="w-40">
                  {commonLanguages.map((language) => (
                    <DropdownMenuItem
                      key={language.code}
                      onClick={() => onTranslate(language.code)}
                      disabled={isTranslating}
                      className={
                        currentTranslationLang === language.code
                          ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                          : ''
                      }
                    >
                      {language.name}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuSubContent>
              </DropdownMenuSub>
            )}

            <DropdownMenuSeparator />

            {onToggleRead && (
              <DropdownMenuItem onClick={onToggleRead}>
                <CheckCheck className="w-4 h-4 mr-2" />
                {email?.is_read ? '标记为未读' : '标记为已读'}
              </DropdownMenuItem>
            )}

            {onToggleStar && (
              <DropdownMenuItem onClick={onToggleStar}>
                <Star
                  className={`w-4 h-4 mr-2 ${email?.is_starred ? 'text-yellow-500 fill-current' : ''}`}
                />
                {email?.is_starred ? '移除星标' : '添加星标'}
              </DropdownMenuItem>
            )}

            {onArchive && (
              <DropdownMenuItem onClick={onArchive}>
                <Archive className="w-4 h-4 mr-2" />
                归档
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      }
    />
  );
}

// 写信页面头部
interface ComposeHeaderProps {
  onSave?: () => void;
  onSend?: () => void;
  onDiscard?: () => void;
}

export function ComposeHeader({ onSave, onSend, onDiscard }: ComposeHeaderProps) {
  return (
    <MobileHeader
      title="写信"
      showBack={true}
      onBack={onDiscard}
      rightContent={
        <div className="flex items-center gap-2">
          {onSave && (
            <Button variant="ghost" size="sm" onClick={onSave} className="text-sm px-3 py-1">
              保存
            </Button>
          )}
          {onSend && (
            <Button
              size="sm"
              onClick={onSend}
              className="text-sm px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white"
            >
              发送
            </Button>
          )}
        </div>
      }
    />
  );
}
