import type { EmailAccount } from '@/types/email';

// 编辑配置类型
export interface AccountEditConfig {
  type: 'oauth2' | 'basic' | 'custom';
  title: string;
  description: string;
  editableFields: string[];
  showReauth?: boolean;
  showPassword?: boolean;
  showImapSmtp?: boolean;
  showOAuth2Config?: boolean;
  providerType?: string;
}

// 根据邮箱账户获取编辑配置
export function getAccountEditConfig(account: EmailAccount): AccountEditConfig {
  const provider = account.provider?.toLowerCase();
  const authMethod = account.auth_method?.toLowerCase();

  // Gmail OAuth2
  if (provider === 'gmail' && authMethod === 'oauth2') {
    return {
      type: 'oauth2',
      title: 'Gmail OAuth2 账户',
      description: '通过Google官方授权的Gmail账户',
      editableFields: ['name', 'is_active'],
      showReauth: true,
      providerType: 'gmail-oauth2',
    };
  }

  // Gmail 应用专用密码
  if (provider === 'gmail' && authMethod === 'password') {
    return {
      type: 'basic',
      title: 'Gmail 应用专用密码',
      description: '使用应用专用密码的Gmail账户',
      editableFields: ['name', 'email', 'password', 'is_active'],
      showPassword: true,
      providerType: 'gmail-password',
    };
  }

  // Outlook OAuth2
  if (provider === 'outlook' && authMethod === 'oauth2') {
    return {
      type: 'oauth2',
      title: 'Outlook OAuth2 账户',
      description: '通过Microsoft官方授权的Outlook账户',
      editableFields: ['name', 'is_active'],
      showReauth: true,
      providerType: 'outlook-oauth2',
    };
  }

  // Outlook 手动OAuth2配置
  if (provider === 'outlook' && authMethod === 'oauth2_manual') {
    return {
      type: 'oauth2',
      title: 'Outlook 手动OAuth2',
      description: '手动配置OAuth2参数的Outlook账户',
      editableFields: ['name', 'email', 'client_id', 'client_secret', 'refresh_token', 'is_active'],
      showOAuth2Config: true,
      providerType: 'outlook-manual',
    };
  }

  // QQ邮箱
  if (provider === 'qq') {
    return {
      type: 'basic',
      title: 'QQ邮箱',
      description: '使用授权码的QQ邮箱账户',
      editableFields: ['name', 'email', 'password', 'is_active'],
      showPassword: true,
      providerType: 'qq',
    };
  }

  // 163邮箱
  if (provider === '163' || provider === 'netease') {
    return {
      type: 'basic',
      title: '163邮箱',
      description: '使用客户端授权码的163邮箱账户',
      editableFields: ['name', 'email', 'password', 'is_active'],
      showPassword: true,
      providerType: '163',
    };
  }

  // 自定义邮箱
  if (provider === 'custom') {
    return {
      type: 'custom',
      title: '自定义邮箱',
      description: '自定义IMAP/SMTP配置的邮箱账户',
      editableFields: [
        'name',
        'email',
        'username',
        'password',
        'imap_host',
        'imap_port',
        'imap_security',
        'smtp_host',
        'smtp_port',
        'smtp_security',
        'is_active',
      ],
      showPassword: true,
      showImapSmtp: true,
      providerType: 'custom',
    };
  }

  // 默认配置（兼容旧数据）
  return {
    type: 'custom',
    title: '邮箱账户',
    description: '邮箱账户设置',
    editableFields: [
      'name',
      'password',
      'imap_host',
      'imap_port',
      'imap_security',
      'smtp_host',
      'smtp_port',
      'smtp_security',
      'is_active',
    ],
    showPassword: true,
    showImapSmtp: true,
    providerType: 'unknown',
  };
}
