'use client';

import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';
import type { EmailAccount } from '@/types/email';
import { AccountEditForm } from '@/components/account-edit/account-edit-form';

interface AccountSettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  account: EmailAccount | null;
}

export function AccountSettingsModal({ isOpen, onClose, account }: AccountSettingsModalProps) {
  const { updateAccount } = useMailboxStore();

  const handleSuccess = () => {
    toast.success('账户设置已更新');
    onClose();
  };

  const handleClose = () => {
    onClose();
  };

  if (!isOpen || !account) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-black/50" onClick={handleClose} />

      {/* 弹窗内容 */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col mx-4">
        {/* 头部 */}
        <div className="flex-shrink-0 flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            账户设置 - {account.email}
          </h2>
          <Button variant="ghost" size="sm" onClick={handleClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>

        {/* 表单内容 */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          <AccountEditForm
            account={account}
            onSuccess={handleSuccess}
            onCancel={handleClose}
            updateAccount={updateAccount}
          />
        </div>
      </div>
    </div>
  );
}
