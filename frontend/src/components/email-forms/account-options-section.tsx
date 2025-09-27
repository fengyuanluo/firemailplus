'use client';

import { useEffect } from 'react';
import { ProxyConfigFields } from '@/components/proxy-config';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import type { UseFormReturn } from 'react-hook-form';

interface AccountOptionsSectionProps {
  form: UseFormReturn<any>;
  disabled?: boolean;
  groupFieldName?: string;
  compactProxy?: boolean;
}

export function AccountOptionsSection({
  form,
  disabled = false,
  groupFieldName = 'group_id',
  compactProxy = false,
}: AccountOptionsSectionProps) {
  const { accountGroups, setAccountGroups } = useMailboxStore();
  useEffect(() => {
    if (accountGroups.length === 0) {
      apiClient
        .getAccountGroups()
        .then((response) => {
          if (response.success && response.data) {
            setAccountGroups(response.data);
          }
        })
        .catch((error) => {
          console.error('Failed to load account groups:', error);
        });
    }
  }, [accountGroups.length, setAccountGroups]);
  const groupValue = form.watch(groupFieldName) ?? '';

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Label className="text-gray-700 dark:text-gray-300">邮箱分组</Label>
        <Select
          value={groupValue}
          onValueChange={(value) => form.setValue(groupFieldName, value)}
          disabled={disabled}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="选择分组（未分组将显示在默认区域）" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">未分组</SelectItem>
            {accountGroups.map((group) => (
              <SelectItem key={group.id} value={group.id.toString()}>
                {group.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          未分组时，该邮箱会显示在默认列表中。可随时在侧边栏拖动调整所属分组。
        </p>
      </div>

      <ProxyConfigFields form={form} disabled={disabled} compact={compactProxy} />
    </div>
  );
}
