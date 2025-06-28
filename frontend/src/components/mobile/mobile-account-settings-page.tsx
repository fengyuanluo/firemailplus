'use client';

import { useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { MobileLayout, MobilePage, MobileContent } from './mobile-layout';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';
import { AccountEditForm } from '@/components/account-edit/account-edit-form';

interface MobileAccountSettingsPageProps {
  accountId: number;
}

export function MobileAccountSettingsPage({ accountId }: MobileAccountSettingsPageProps) {
  const router = useRouter();
  const { accounts, updateAccount } = useMailboxStore();

  const account = accounts.find((acc) => acc.id === accountId);

  const handleSuccess = () => {
    toast.success('账户设置已更新');
    router.back();
  };

  const handleCancel = () => {
    router.back();
  };

  if (!account) {
    return (
      <MobileLayout>
        <MobilePage>
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <div className="text-gray-500 dark:text-gray-400">账户不存在</div>
              <Button onClick={() => router.back()} className="mt-4">
                返回
              </Button>
            </div>
          </div>
        </MobilePage>
      </MobileLayout>
    );
  }

  return (
    <MobileLayout>
      <MobilePage>
        {/* 头部 */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" onClick={handleCancel}>
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">账户设置</h1>
          </div>
        </div>

        {/* 表单内容 */}
        <MobileContent>
          <AccountEditForm
            account={account}
            onSuccess={handleSuccess}
            onCancel={handleCancel}
            updateAccount={updateAccount}
          />
        </MobileContent>
      </MobilePage>
    </MobileLayout>
  );
}
