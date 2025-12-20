import { useCallback, useEffect, useMemo, useState } from 'react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import type { EmailGroup } from '@/types/email';
import { Loader2 } from 'lucide-react';

interface EmailGroupSelectorProps {
  value?: number | null;
  onChange: (groupId: number | null) => void;
  disabled?: boolean;
  label?: string;
  placeholder?: string;
  autoSelectDefault?: boolean;
}

export function EmailGroupSelector({
  value,
  onChange,
  disabled = false,
  label = '分组',
  placeholder = '选择分组（可选）',
  autoSelectDefault = true,
}: EmailGroupSelectorProps) {
  const { groups, setGroups } = useMailboxStore();
  const [loading, setLoading] = useState(false);

  const loadGroups = useCallback(async () => {
    setLoading(true);
    try {
      const response = await apiClient.getEmailGroups();
      if (response.success && response.data) {
        setGroups(response.data);
      }
    } catch (error) {
      console.error('Failed to load email groups:', error);
    } finally {
      setLoading(false);
    }
  }, [setGroups]);

  useEffect(() => {
    if (groups.length === 0) {
      loadGroups();
    }
  }, [groups.length, loadGroups]);

  const defaultGroupId = useMemo(
    () => groups.find((g) => g.is_default)?.id,
    [groups]
  );

  useEffect(() => {
    if (autoSelectDefault && value == null && defaultGroupId) {
      onChange(defaultGroupId);
    }
  }, [autoSelectDefault, defaultGroupId, onChange, value]);

  const sortedGroups: EmailGroup[] = useMemo(() => {
    return [...groups].sort((a, b) => {
      if (a.is_default && !b.is_default) return -1;
      if (!a.is_default && b.is_default) return 1;
      return a.sort_order - b.sort_order;
    });
  }, [groups]);

  const resolvedValue = value ?? (autoSelectDefault ? defaultGroupId ?? null : null);

  return (
    <div className="space-y-2">
      <Label className="text-gray-700 dark:text-gray-300">{label}</Label>
      <Select
        value={resolvedValue ? String(resolvedValue) : ''}
        onValueChange={(val) => onChange(val ? Number(val) : null)}
        disabled={disabled || loading}
      >
        <SelectTrigger className="w-full h-10 border-gray-300 dark:border-gray-600">
          {loading ? (
            <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <Loader2 className="w-4 h-4 animate-spin" />
              加载分组...
            </div>
          ) : (
            <SelectValue placeholder={placeholder} />
          )}
        </SelectTrigger>
        <SelectContent>
          {sortedGroups.map((group) => (
            <SelectItem key={group.id} value={String(group.id)}>
              <div className="flex items-center justify-between w-full">
                <span>{group.name}</span>
                <span className="text-xs text-gray-400">
                  {group.is_default ? '默认' : ''} {group.account_count > 0 ? `(${group.account_count})` : ''}
                </span>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
