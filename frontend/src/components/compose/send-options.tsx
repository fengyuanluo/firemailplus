'use client';

import { useState } from 'react';
import { Settings, Clock, Eye, Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';

export interface SendOptionsData {
  priority: 'low' | 'normal' | 'high';
  scheduledTime?: string;
  requestReadReceipt: boolean;
  requestDeliveryReceipt: boolean;
  importance: 'low' | 'normal' | 'high';
}

interface SendOptionsProps {
  options: SendOptionsData;
  onChange: (options: SendOptionsData) => void;
  onSend: () => void;
  onPreview: () => void;
  onSchedule: (scheduledTime: string) => void;
  isSending?: boolean;
}

export function SendOptions({
  options,
  onChange,
  onSend,
  onPreview,
  onSchedule,
  isSending = false,
}: SendOptionsProps) {
  const [showScheduleDialog, setShowScheduleDialog] = useState(false);
  const [scheduledDateTime, setScheduledDateTime] = useState('');

  // 优先级选项
  const priorityOptions = [
    { value: 'low', label: '低优先级', icon: '🔽' },
    { value: 'normal', label: '普通', icon: '➖' },
    { value: 'high', label: '高优先级', icon: '🔺' },
  ];

  // 重要性选项
  const importanceOptions = [
    { value: 'low', label: '低重要性', icon: '🔽' },
    { value: 'normal', label: '普通', icon: '➖' },
    { value: 'high', label: '高重要性', icon: '🔺' },
  ];

  // 快速时间选项
  const quickTimeOptions = [
    { label: '1小时后', value: 1 },
    { label: '明天上午9点', value: 'tomorrow_9am' },
    { label: '下周一上午9点', value: 'next_monday_9am' },
  ];

  // 计算快速时间
  const calculateQuickTime = (option: string | number): string => {
    const now = new Date();

    if (typeof option === 'number') {
      // 小时后
      const future = new Date(now.getTime() + option * 60 * 60 * 1000);
      return future.toISOString().slice(0, 16);
    }

    switch (option) {
      case 'tomorrow_9am':
        const tomorrow = new Date(now);
        tomorrow.setDate(tomorrow.getDate() + 1);
        tomorrow.setHours(9, 0, 0, 0);
        return tomorrow.toISOString().slice(0, 16);

      case 'next_monday_9am':
        const nextMonday = new Date(now);
        const daysUntilMonday = (8 - nextMonday.getDay()) % 7 || 7;
        nextMonday.setDate(nextMonday.getDate() + daysUntilMonday);
        nextMonday.setHours(9, 0, 0, 0);
        return nextMonday.toISOString().slice(0, 16);

      default:
        return '';
    }
  };

  // 处理定时发送
  const handleScheduleSend = () => {
    if (scheduledDateTime) {
      onSchedule(scheduledDateTime);
      setShowScheduleDialog(false);
      setScheduledDateTime('');
    }
  };

  // 格式化显示时间
  const formatScheduledTime = (time: string) => {
    const date = new Date(time);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffHours = Math.round(diffMs / (1000 * 60 * 60));

    if (diffHours < 24) {
      return `${diffHours}小时后`;
    } else {
      const diffDays = Math.round(diffHours / 24);
      return `${diffDays}天后`;
    }
  };

  return (
    <div className="flex items-center gap-2">
      {/* 预览按钮 */}
      <Button variant="outline" size="sm" onClick={onPreview} className="gap-2">
        <Eye className="w-4 h-4" />
        预览
      </Button>

      {/* 发送选项 */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            <Settings className="w-4 h-4" />
            选项
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent className="w-80 p-4">
          <DropdownMenuLabel>邮件发送选项</DropdownMenuLabel>
          <DropdownMenuSeparator />

          <div className="space-y-4">
            {/* 优先级设置 */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">优先级</Label>
              <Select
                value={options.priority}
                onValueChange={(value: 'low' | 'normal' | 'high') =>
                  onChange({ ...options, priority: value })
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {priorityOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      <div className="flex items-center gap-2">
                        <span>{option.icon}</span>
                        <span>{option.label}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* 重要性设置 */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">重要性</Label>
              <Select
                value={options.importance}
                onValueChange={(value: 'low' | 'normal' | 'high') =>
                  onChange({ ...options, importance: value })
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {importanceOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      <div className="flex items-center gap-2">
                        <span>{option.icon}</span>
                        <span>{option.label}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* 回执设置 */}
            <div className="space-y-3">
              <Label className="text-sm font-medium">回执设置</Label>

              <div className="flex items-center justify-between">
                <Label htmlFor="read-receipt" className="text-sm">
                  请求已读回执
                </Label>
                <Switch
                  id="read-receipt"
                  checked={options.requestReadReceipt}
                  onCheckedChange={(checked) =>
                    onChange({ ...options, requestReadReceipt: checked })
                  }
                />
              </div>

              <div className="flex items-center justify-between">
                <Label htmlFor="delivery-receipt" className="text-sm">
                  请求送达回执
                </Label>
                <Switch
                  id="delivery-receipt"
                  checked={options.requestDeliveryReceipt}
                  onCheckedChange={(checked) =>
                    onChange({ ...options, requestDeliveryReceipt: checked })
                  }
                />
              </div>
            </div>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* 定时发送 */}
      <DropdownMenu open={showScheduleDialog} onOpenChange={setShowScheduleDialog}>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            <Clock className="w-4 h-4" />
            {options.scheduledTime ? formatScheduledTime(options.scheduledTime) : '定时'}
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent className="w-72 p-4">
          <DropdownMenuLabel>定时发送</DropdownMenuLabel>
          <DropdownMenuSeparator />

          <div className="space-y-4">
            {/* 快速选择 */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">快速选择</Label>
              <div className="grid grid-cols-1 gap-1">
                {quickTimeOptions.map((option) => (
                  <Button
                    key={option.label}
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      const time = calculateQuickTime(option.value);
                      setScheduledDateTime(time);
                    }}
                    className="justify-start text-sm"
                  >
                    {option.label}
                  </Button>
                ))}
              </div>
            </div>

            {/* 自定义时间 */}
            <div className="space-y-2">
              <Label htmlFor="custom-time" className="text-sm font-medium">
                自定义时间
              </Label>
              <Input
                id="custom-time"
                type="datetime-local"
                value={scheduledDateTime}
                onChange={(e) => setScheduledDateTime(e.target.value)}
                min={new Date().toISOString().slice(0, 16)}
              />
            </div>

            {/* 操作按钮 */}
            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={handleScheduleSend}
                disabled={!scheduledDateTime}
                className="flex-1"
              >
                确定
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setShowScheduleDialog(false);
                  setScheduledDateTime('');
                }}
                className="flex-1"
              >
                取消
              </Button>
            </div>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* 立即发送按钮 */}
      <Button
        onClick={onSend}
        disabled={isSending}
        className="gap-2 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900"
      >
        <Send className="w-4 h-4" />
        {isSending ? '发送中...' : '发送'}
      </Button>
    </div>
  );
}
