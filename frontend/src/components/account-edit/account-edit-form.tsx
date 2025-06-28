'use client';

import type { EmailAccount } from '@/types/email';
import { getAccountEditConfig } from './account-edit-config';
import { BasicEditForm } from './basic-edit-form';
import { OAuth2EditForm } from './oauth2-edit-form';
import { CustomEditForm } from './custom-edit-form';

interface AccountEditFormProps {
  account: EmailAccount;
  onSuccess: () => void;
  onCancel: () => void;
  updateAccount: (account: EmailAccount) => void;
}

export function AccountEditForm({
  account,
  onSuccess,
  onCancel,
  updateAccount,
}: AccountEditFormProps) {
  const config = getAccountEditConfig(account);

  // 根据配置类型渲染不同的编辑表单
  switch (config.type) {
    case 'oauth2':
      return (
        <OAuth2EditForm
          account={account}
          config={config}
          onSuccess={onSuccess}
          onCancel={onCancel}
          updateAccount={updateAccount}
        />
      );

    case 'custom':
      return (
        <CustomEditForm
          account={account}
          config={config}
          onSuccess={onSuccess}
          onCancel={onCancel}
          updateAccount={updateAccount}
        />
      );

    case 'basic':
    default:
      return (
        <BasicEditForm
          account={account}
          config={config}
          onSuccess={onSuccess}
          onCancel={onCancel}
          updateAccount={updateAccount}
        />
      );
  }
}
