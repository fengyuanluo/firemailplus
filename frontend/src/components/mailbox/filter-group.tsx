'use client';

import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';

interface FilterOption {
  id: string;
  label: string;
  count?: number;
  checked?: boolean;
}

interface FilterGroupProps {
  title: string;
  options: FilterOption[];
  maxVisible?: number;
  onOptionChange: (optionId: string, checked: boolean) => void;
  onClearAll?: () => void;
}

export function FilterGroup({
  title,
  options,
  maxVisible = 10,
  onOptionChange,
  onClearAll,
}: FilterGroupProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [showAll, setShowAll] = useState(false);

  const visibleOptions = showAll ? options : options.slice(0, maxVisible);
  const hasMore = options.length > maxVisible;
  const checkedCount = options.filter((option) => option.checked).length;

  return (
    <div className="border-b border-gray-200 dark:border-gray-700 pb-4 mb-4">
      {/* 分组标题 */}
      <div className="flex items-center justify-between mb-3">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsExpanded(!isExpanded)}
          className="p-0 h-auto font-medium text-gray-900 dark:text-gray-100 hover:bg-transparent"
        >
          <div className="flex items-center gap-2">
            {isExpanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )}
            <span>{title}</span>
            {checkedCount > 0 && (
              <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded-full">
                {checkedCount}
              </span>
            )}
          </div>
        </Button>

        {checkedCount > 0 && onClearAll && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onClearAll}
            className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 p-1 h-auto"
          >
            清除
          </Button>
        )}
      </div>

      {/* 选项列表 */}
      {isExpanded && (
        <div className="space-y-2">
          {visibleOptions.map((option) => (
            <div key={option.id} className="flex items-center space-x-2">
              <Checkbox
                id={option.id}
                checked={option.checked || false}
                onCheckedChange={(checked) => onOptionChange(option.id, checked as boolean)}
                className="data-[state=checked]:bg-blue-600 data-[state=checked]:border-blue-600"
              />
              <label
                htmlFor={option.id}
                className="flex-1 text-sm text-gray-700 dark:text-gray-300 cursor-pointer flex items-center justify-between"
              >
                <span className="truncate">{option.label}</span>
                {option.count !== undefined && (
                  <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                    {option.count}
                  </span>
                )}
              </label>
            </div>
          ))}

          {/* 显示更多/收起按钮 */}
          {hasMore && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowAll(!showAll)}
              className="w-full text-xs text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 p-1 h-auto"
            >
              {showAll ? '收起' : `显示更多 (${options.length - maxVisible})`}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

// 日期范围筛选组件
interface DateRangeFilterProps {
  title: string;
  value?: { start?: string; end?: string };
  onChange: (range: { start?: string; end?: string }) => void;
  onClear?: () => void;
}

export function DateRangeFilter({ title, value, onChange, onClear }: DateRangeFilterProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [selectedPreset, setSelectedPreset] = useState<string>('');

  const presets = [
    { id: 'today', label: '今天', days: 0 },
    { id: 'yesterday', label: '昨天', days: 1 },
    { id: 'week', label: '本周', days: 7 },
    { id: 'month', label: '本月', days: 30 },
    { id: 'quarter', label: '本季度', days: 90 },
    { id: 'year', label: '本年', days: 365 },
  ];

  const handlePresetChange = (presetId: string, days: number) => {
    setSelectedPreset(presetId);

    const now = new Date();
    const start = new Date(now);
    start.setDate(start.getDate() - days);

    onChange({
      start: start.toISOString().split('T')[0],
      end: now.toISOString().split('T')[0],
    });
  };

  const handleClear = () => {
    setSelectedPreset('');
    onChange({});
    onClear?.();
  };

  const hasValue = value?.start || value?.end;

  return (
    <div className="border-b border-gray-200 dark:border-gray-700 pb-4 mb-4">
      {/* 分组标题 */}
      <div className="flex items-center justify-between mb-3">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsExpanded(!isExpanded)}
          className="p-0 h-auto font-medium text-gray-900 dark:text-gray-100 hover:bg-transparent"
        >
          <div className="flex items-center gap-2">
            {isExpanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )}
            <span>{title}</span>
            {hasValue && (
              <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded-full">
                已选择
              </span>
            )}
          </div>
        </Button>

        {hasValue && (
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClear}
            className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 p-1 h-auto"
          >
            清除
          </Button>
        )}
      </div>

      {/* 日期选项 */}
      {isExpanded && (
        <div className="space-y-2">
          {/* 预设时间范围 */}
          {presets.map((preset) => (
            <div key={preset.id} className="flex items-center space-x-2">
              <Checkbox
                id={preset.id}
                checked={selectedPreset === preset.id}
                onCheckedChange={(checked) => {
                  if (checked) {
                    handlePresetChange(preset.id, preset.days);
                  } else {
                    handleClear();
                  }
                }}
                className="data-[state=checked]:bg-blue-600 data-[state=checked]:border-blue-600"
              />
              <label
                htmlFor={preset.id}
                className="text-sm text-gray-700 dark:text-gray-300 cursor-pointer"
              >
                {preset.label}
              </label>
            </div>
          ))}

          {/* 自定义日期范围 */}
          <div className="mt-4 space-y-2">
            <div className="text-xs text-gray-500 dark:text-gray-400">自定义范围</div>
            <div className="grid grid-cols-2 gap-2">
              <input
                type="date"
                value={value?.start || ''}
                onChange={(e) => onChange({ ...value, start: e.target.value })}
                className="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                placeholder="开始日期"
              />
              <input
                type="date"
                value={value?.end || ''}
                onChange={(e) => onChange({ ...value, end: e.target.value })}
                className="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                placeholder="结束日期"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
