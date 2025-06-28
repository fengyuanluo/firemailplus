'use client';

import { Inbox, Edit } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useComposeStore, useMailboxStore } from '@/lib/store';

export function SidebarHeader() {
  const { openCompose } = useComposeStore();
  const { selectFolder, selectAccount } = useMailboxStore();

  const handleCompose = () => {
    openCompose();
  };

  const handleAllInbox = () => {
    // 清除账户和文件夹选择，显示所有邮件
    selectAccount(null);
    selectFolder(null);
    // 邮件列表会自动重新加载（通过 EmailList 组件的 useEffect）
  };

  return (
    <div className="flex-shrink-0 p-4 border-b border-gray-200 dark:border-gray-700">
      <div className="space-y-2">
        {/* 全部收件按钮 */}
        <Button
          onClick={handleAllInbox}
          variant="outline"
          className="w-full justify-start gap-2 h-10 border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700"
        >
          <Inbox className="w-4 h-4" />
          全部收件
        </Button>

        {/* 写信按钮 */}
        <Button
          onClick={handleCompose}
          className="w-full justify-start gap-2 h-10 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900"
        >
          <Edit className="w-4 h-4" />
          写信
        </Button>
      </div>
    </div>
  );
}
