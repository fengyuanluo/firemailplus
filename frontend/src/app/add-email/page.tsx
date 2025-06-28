'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ProtectedRoute } from '@/components/auth/route-guard';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { ArrowLeft } from 'lucide-react';
import { GmailOAuth2Form } from '@/components/email-forms/gmail-oauth2-form';
import { GmailPasswordForm } from '@/components/email-forms/gmail-password-form';
import { OutlookOAuth2Form } from '@/components/email-forms/outlook-oauth2-form';
import { OutlookManualOAuth2Form } from '@/components/email-forms/outlook-manual-oauth2-form';
import { OutlookBatchForm } from '@/components/email-forms/outlook-batch-form';
import { QQForm } from '@/components/email-forms/qq-form';
import { NetEaseForm } from '@/components/email-forms/netease-form';
import { CustomForm } from '@/components/email-forms/custom-form';

// 邮箱类型定义
export type EmailProviderType =
  | 'gmail-oauth2'
  | 'gmail-password'
  | 'outlook-oauth2'
  | 'outlook-manual'
  | 'outlook-batch'
  | 'qq'
  | '163'
  | 'custom';

interface EmailProvider {
  id: EmailProviderType;
  name: string;
  description: string;
  category: 'oauth2' | 'password' | 'custom' | 'batch';
}

const EMAIL_PROVIDERS: EmailProvider[] = [
  {
    id: 'gmail-oauth2',
    name: 'Gmail (OAuth2)',
    description: 'Google 官方授权，安全便捷',
    category: 'oauth2',
  },
  {
    id: 'gmail-password',
    name: 'Gmail (应用专用密码)',
    description: '使用应用专用密码连接',
    category: 'password',
  },
  {
    id: 'outlook-oauth2',
    name: 'Outlook (OAuth2)',
    description: 'Microsoft 官方授权，安全便捷',
    category: 'oauth2',
  },
  {
    id: 'outlook-manual',
    name: 'Outlook (手动配置)',
    description: '手动配置 OAuth2 参数',
    category: 'oauth2',
  },
  {
    id: 'outlook-batch',
    name: 'Outlook (批量添加)',
    description: '批量导入多个 Outlook 账户',
    category: 'batch',
  },
  {
    id: 'qq',
    name: 'QQ邮箱',
    description: '使用授权码连接',
    category: 'password',
  },
  {
    id: '163',
    name: '163邮箱',
    description: '使用授权码连接',
    category: 'password',
  },
  {
    id: 'custom',
    name: '自定义邮箱',
    description: '手动配置 IMAP/SMTP 服务器',
    category: 'custom',
  },
];

export default function AddEmailPage() {
  const router = useRouter();
  const [selectedProvider, setSelectedProvider] = useState<EmailProviderType | null>(null);

  const handleProviderSelect = (providerId: string) => {
    setSelectedProvider(providerId as EmailProviderType);
  };

  const handleBack = () => {
    router.back();
  };

  const handleFormSuccess = () => {
    // 成功添加邮箱后跳转到邮箱页面
    router.push('/mailbox');
  };

  const handleFormCancel = () => {
    setSelectedProvider(null);
  };

  const renderProviderForm = () => {
    if (!selectedProvider) return null;

    switch (selectedProvider) {
      case 'gmail-oauth2':
        return <GmailOAuth2Form onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      case 'gmail-password':
        return <GmailPasswordForm onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      case 'outlook-oauth2':
        return <OutlookOAuth2Form onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      case 'outlook-manual':
        return (
          <OutlookManualOAuth2Form onSuccess={handleFormSuccess} onCancel={handleFormCancel} />
        );

      case 'outlook-batch':
        return <OutlookBatchForm onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      case 'qq':
        return <QQForm onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      case '163':
        return <NetEaseForm onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      case 'custom':
        return <CustomForm onSuccess={handleFormSuccess} onCancel={handleFormCancel} />;

      default:
        return (
          <Card className="border border-gray-200 dark:border-gray-700">
            <CardContent className="pt-6">
              <div className="text-center text-gray-500 dark:text-gray-400">
                {selectedProvider} 配置表单开发中...
              </div>
            </CardContent>
          </Card>
        );
    }
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 py-8 px-4 sm:px-6 lg:px-8">
        <div className="max-w-2xl mx-auto">
          {/* 头部 */}
          <div className="flex items-center gap-4 mb-8">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleBack}
              className="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
            >
              <ArrowLeft className="w-4 h-4 mr-2" />
              返回
            </Button>
            <h1 className="text-2xl font-light text-gray-900 dark:text-gray-100">添加邮箱账户</h1>
          </div>

          {/* 邮箱类型选择 */}
          <Card className="border border-gray-200 dark:border-gray-700 mb-6">
            <CardHeader>
              <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
                选择邮箱类型
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Select onValueChange={handleProviderSelect}>
                <SelectTrigger className="w-full h-12 border-gray-300 dark:border-gray-600">
                  <SelectValue placeholder="请选择要添加的邮箱类型">
                    {selectedProvider && (
                      <span className="font-medium">
                        {EMAIL_PROVIDERS.find((p) => p.id === selectedProvider)?.name}
                      </span>
                    )}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {EMAIL_PROVIDERS.map((provider) => (
                    <SelectItem key={provider.id} value={provider.id}>
                      <div className="flex flex-col py-1">
                        <span className="font-medium">{provider.name}</span>
                        <span className="text-sm text-gray-500 dark:text-gray-400">
                          {provider.description}
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </CardContent>
          </Card>

          {/* 配置表单 */}
          {renderProviderForm()}
        </div>
      </div>
    </ProtectedRoute>
  );
}
