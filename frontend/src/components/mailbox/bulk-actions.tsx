'use client';

import { useState } from 'react';
import { CheckCheck, Star, Trash2, MoreHorizontal, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMailboxStore } from '@/lib/store';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';

interface BulkActionsProps {
  selectedCount: number;
}

export function BulkActions({ selectedCount }: BulkActionsProps) {
  const [isOperating, setIsOperating] = useState(false);

  const { selectedEmails, clearSelection, updateEmail, removeEmail } = useMailboxStore();

  // 执行批量操作
  const executeBulkOperation = async (operation: string) => {
    if (selectedEmails.size === 0) return;

    setIsOperating(true);

    try {
      const emailIds = Array.from(selectedEmails);

      const response = await apiClient.batchEmailOperation({
        email_ids: emailIds,
        operation,
      });

      if (response.success) {
        // 乐观更新本地状态
        switch (operation) {
          case 'read':
            emailIds.forEach((id) => updateEmail(id, { is_read: true }));
            toast.success(`已标记 ${selectedCount} 封邮件为已读`);
            break;
          case 'unread':
            emailIds.forEach((id) => updateEmail(id, { is_read: false }));
            toast.success(`已标记 ${selectedCount} 封邮件为未读`);
            break;
          case 'star':
            emailIds.forEach((id) => updateEmail(id, { is_starred: true }));
            toast.success(`已为 ${selectedCount} 封邮件添加星标`);
            break;
          case 'unstar':
            emailIds.forEach((id) => updateEmail(id, { is_starred: false }));
            toast.success(`已移除 ${selectedCount} 封邮件的星标`);
            break;
          case 'delete':
            emailIds.forEach((id) => removeEmail(id));
            toast.success(`已删除 ${selectedCount} 封邮件`);
            break;
        }

        clearSelection();
      } else {
        toast.error(response.message || '批量操作失败');
      }
    } catch (error: any) {
      toast.error(error.message || '批量操作失败');
    } finally {
      setIsOperating(false);
    }
  };

  if (selectedCount === 0) return null;

  return (
    <div className="flex items-center gap-1">
      {/* 标记已读 */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => executeBulkOperation('read')}
        disabled={isOperating}
        className="p-2"
        title="标记为已读"
      >
        {isOperating ? (
          <Loader2 className="w-4 h-4 animate-spin" />
        ) : (
          <CheckCheck className="w-4 h-4" />
        )}
      </Button>

      {/* 添加星标 */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => executeBulkOperation('star')}
        disabled={isOperating}
        className="p-2"
        title="添加星标"
      >
        <Star className="w-4 h-4" />
      </Button>

      {/* 删除 */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => {
          if (confirm(`确定要删除这 ${selectedCount} 封邮件吗？`)) {
            executeBulkOperation('delete');
          }
        }}
        disabled={isOperating}
        className="p-2 text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
        title="删除"
      >
        <Trash2 className="w-4 h-4" />
      </Button>

      {/* 更多操作 */}
      <Button variant="ghost" size="sm" disabled={isOperating} className="p-2" title="更多操作">
        <MoreHorizontal className="w-4 h-4" />
      </Button>
    </div>
  );
}
