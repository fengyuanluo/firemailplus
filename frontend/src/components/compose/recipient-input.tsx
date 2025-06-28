'use client';

import { useState, useRef } from 'react';
import { X, User, Users } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

interface Recipient {
  email: string;
  name?: string;
  isValid: boolean;
}

interface RecipientInputProps {
  label: string;
  placeholder: string;
  recipients: Recipient[];
  onChange: (recipients: Recipient[]) => void;
  showContactPicker?: boolean;
  maxRecipients?: number;
}

export function RecipientInput({
  label,
  placeholder,
  recipients,
  onChange,
  showContactPicker = false,
  maxRecipients = 100,
}: RecipientInputProps) {
  const [inputValue, setInputValue] = useState('');
  const [suggestions, setSuggestions] = useState<Recipient[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [focusedSuggestionIndex, setFocusedSuggestionIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  // 邮箱地址验证
  const validateEmail = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  };

  // 解析输入的邮箱地址
  const parseEmailInput = (input: string): Recipient[] => {
    const emails = input.split(/[,;\s]+/).filter((email) => email.trim());
    return emails.map((email) => {
      const trimmedEmail = email.trim();
      // 支持 "Name <email@domain.com>" 格式
      const match = trimmedEmail.match(/^(.+?)\s*<(.+?)>$|^(.+)$/);
      if (match) {
        const name = match[1]?.trim();
        const emailAddr = match[2] || match[3];
        return {
          email: emailAddr,
          name: name || undefined,
          isValid: validateEmail(emailAddr),
        };
      }
      return {
        email: trimmedEmail,
        isValid: validateEmail(trimmedEmail),
      };
    });
  };

  // 获取邮箱建议
  const getSuggestions = async (query: string): Promise<Recipient[]> => {
    if (!query || query.length < 2) return [];

    // TODO: 调用API获取联系人建议
    // 这里使用模拟数据
    const mockSuggestions: Recipient[] = [
      { email: 'john.doe@example.com', name: 'John Doe', isValid: true },
      { email: 'jane.smith@company.com', name: 'Jane Smith', isValid: true },
      { email: 'admin@domain.com', name: 'Admin', isValid: true },
    ];

    return mockSuggestions.filter(
      (suggestion) =>
        suggestion.email.toLowerCase().includes(query.toLowerCase()) ||
        suggestion.name?.toLowerCase().includes(query.toLowerCase())
    );
  };

  // 处理输入变化
  const handleInputChange = async (value: string) => {
    setInputValue(value);

    if (value.trim()) {
      const suggestions = await getSuggestions(value);
      setSuggestions(suggestions);
      setShowSuggestions(suggestions.length > 0);
      setFocusedSuggestionIndex(-1);
    } else {
      setShowSuggestions(false);
    }
  };

  // 添加收件人
  const addRecipient = (recipient: Recipient) => {
    if (recipients.length >= maxRecipients) return;

    // 检查是否已存在
    const exists = recipients.some((r) => r.email === recipient.email);
    if (!exists) {
      onChange([...recipients, recipient]);
    }

    setInputValue('');
    setShowSuggestions(false);
    inputRef.current?.focus();
  };

  // 删除收件人
  const removeRecipient = (index: number) => {
    const newRecipients = recipients.filter((_, i) => i !== index);
    onChange(newRecipients);
  };

  // 处理键盘事件
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (inputValue.trim()) {
        const newRecipients = parseEmailInput(inputValue);
        const validRecipients = newRecipients.filter((r) => r.isValid);

        validRecipients.forEach((recipient) => {
          addRecipient(recipient);
        });
      }
    } else if (e.key === 'Backspace' && !inputValue && recipients.length > 0) {
      removeRecipient(recipients.length - 1);
    } else if (e.key === 'ArrowDown' && showSuggestions) {
      e.preventDefault();
      setFocusedSuggestionIndex((prev) => (prev < suggestions.length - 1 ? prev + 1 : 0));
    } else if (e.key === 'ArrowUp' && showSuggestions) {
      e.preventDefault();
      setFocusedSuggestionIndex((prev) => (prev > 0 ? prev - 1 : suggestions.length - 1));
    } else if (e.key === 'Enter' && focusedSuggestionIndex >= 0) {
      e.preventDefault();
      addRecipient(suggestions[focusedSuggestionIndex]);
    } else if (e.key === 'Escape') {
      setShowSuggestions(false);
      setFocusedSuggestionIndex(-1);
    }
  };

  // 格式化收件人显示
  const formatRecipient = (recipient: Recipient) => {
    return recipient.name ? `${recipient.name} <${recipient.email}>` : recipient.email;
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</label>
        {showContactPicker && (
          <Button variant="ghost" size="sm" className="text-xs gap-1">
            <Users className="w-3 h-3" />
            选择联系人
          </Button>
        )}
      </div>

      <div className="relative">
        {/* 收件人标签和输入框 */}
        <div className="min-h-[40px] p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 flex flex-wrap items-center gap-1">
          {/* 收件人标签 */}
          {recipients.map((recipient, index) => (
            <Badge
              key={index}
              variant={recipient.isValid ? 'default' : 'destructive'}
              className="flex items-center gap-1 max-w-[200px]"
            >
              <User className="w-3 h-3" />
              <span className="truncate text-xs">{formatRecipient(recipient)}</span>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => removeRecipient(index)}
                className="p-0 h-auto w-4 hover:bg-transparent"
              >
                <X className="w-3 h-3" />
              </Button>
            </Badge>
          ))}

          {/* 输入框 */}
          <Input
            ref={inputRef}
            type="text"
            placeholder={recipients.length === 0 ? placeholder : ''}
            value={inputValue}
            onChange={(e) => handleInputChange(e.target.value)}
            onKeyDown={handleKeyDown}
            onFocus={() => inputValue && setShowSuggestions(suggestions.length > 0)}
            onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
            className="flex-1 min-w-[120px] border-0 p-0 h-auto focus:ring-0 focus:outline-none bg-transparent"
          />
        </div>

        {/* 建议列表 */}
        {showSuggestions && suggestions.length > 0 && (
          <div className="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg z-50 max-h-48 overflow-y-auto">
            {suggestions.map((suggestion, index) => (
              <div
                key={index}
                onClick={() => addRecipient(suggestion)}
                className={`p-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 ${
                  index === focusedSuggestionIndex ? 'bg-gray-100 dark:bg-gray-700' : ''
                }`}
              >
                <div className="flex items-center gap-2">
                  <User className="w-4 h-4 text-gray-400" />
                  <div>
                    <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {suggestion.name || suggestion.email}
                    </div>
                    {suggestion.name && (
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {suggestion.email}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 收件人统计 */}
      {recipients.length > 0 && (
        <div className="text-xs text-gray-500 dark:text-gray-400">
          {recipients.length} 个收件人
          {recipients.some((r) => !r.isValid) && (
            <span className="text-red-500 ml-2">包含无效邮箱地址</span>
          )}
        </div>
      )}
    </div>
  );
}
