'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ChevronDown, ChevronRight, ExternalLink, Eye, EyeOff } from 'lucide-react';
import { useOAuth2 } from '@/hooks/use-oauth';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';

// UUID格式验证
const isValidUUID = (value: string) => {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
  return uuidRegex.test(value);
};

const outlookManualOAuth2Schema = z.object({
  name: z.string().min(1, '请输入账户名称'),
  email: z
    .string()
    .email('请输入有效的Outlook邮箱地址')
    .refine((email) => {
      const outlookDomains = ['@outlook.com', '@hotmail.com', '@live.com', '@msn.com'];
      return outlookDomains.some((domain) => email.endsWith(domain));
    }, '请输入Outlook相关域名的邮箱地址（@outlook.com, @hotmail.com, @live.com, @msn.com）'),
  client_id: z
    .string()
    .min(1, '请输入客户端ID')
    .refine(isValidUUID, '客户端ID必须是有效的UUID格式'),
  client_secret: z.string().optional(),
  refresh_token: z.string().min(1, '请输入刷新令牌'),
  scope: z.string().optional(),
});

type OutlookManualOAuth2Form = z.infer<typeof outlookManualOAuth2Schema>;

interface OutlookManualOAuth2FormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function OutlookManualOAuth2Form({ onSuccess, onCancel }: OutlookManualOAuth2FormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showInstructions, setShowInstructions] = useState(false);
  const [showClientSecret, setShowClientSecret] = useState(false);
  const [showRefreshToken, setShowRefreshToken] = useState(false);
  const { createManualOAuth2Account } = useOAuth2();
  const { addAccount } = useMailboxStore();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    setValue,
  } = useForm<OutlookManualOAuth2Form>({
    resolver: zodResolver(outlookManualOAuth2Schema),
    defaultValues: {
      scope:
        'https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/SMTP.Send offline_access',
    },
  });

  const onSubmit = async (data: OutlookManualOAuth2Form) => {
    setIsSubmitting(true);
    try {
      const response = await createManualOAuth2Account.mutateAsync({
        name: data.name,
        email: data.email,
        provider: 'outlook',
        client_id: data.client_id,
        client_secret: data.client_secret || undefined,
        refresh_token: data.refresh_token,
        scope: data.scope || undefined,
      });

      if (response.success && response.data) {
        addAccount(response.data);
        toast.success('Outlook手动OAuth2账户创建成功');
        reset();
        onSuccess?.();
      }
    } catch (error: any) {
      console.error('Outlook manual OAuth2 configuration failed:', error);
      // 错误已在createManualOAuth2Account中处理
    } finally {
      setIsSubmitting(false);
    }
  };

  const fillDefaultScope = () => {
    setValue(
      'scope',
      'https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/SMTP.Send offline_access'
    );
  };

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Outlook 手动 OAuth2 配置
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          使用自定义OAuth2应用配置连接您的Outlook账户
        </p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="space-y-4">
            <div>
              <Label htmlFor="name" className="text-gray-700 dark:text-gray-300">
                账户名称
              </Label>
              <Input
                id="name"
                type="text"
                placeholder="为此账户设置一个名称"
                {...register('name')}
                className={`mt-1 h-10 ${
                  errors.name
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={isSubmitting}
              />
              {errors.name && <p className="text-sm text-red-500 mt-1">{errors.name.message}</p>}
            </div>

            <div>
              <Label htmlFor="email" className="text-gray-700 dark:text-gray-300">
                Outlook邮箱地址
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="your.email@outlook.com"
                {...register('email')}
                className={`mt-1 h-10 ${
                  errors.email
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={isSubmitting}
              />
              {errors.email && <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>}
            </div>

            <div>
              <Label htmlFor="client_id" className="text-gray-700 dark:text-gray-300">
                客户端ID <span className="text-red-500">*</span>
              </Label>
              <Input
                id="client_id"
                type="text"
                placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                {...register('client_id')}
                className={`mt-1 h-10 ${
                  errors.client_id
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={isSubmitting}
              />
              {errors.client_id && (
                <p className="text-sm text-red-500 mt-1">{errors.client_id.message}</p>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Azure应用注册中的应用程序（客户端）ID
              </p>
            </div>

            <div>
              <Label htmlFor="client_secret" className="text-gray-700 dark:text-gray-300">
                客户端密钥 <span className="text-gray-400">(可选)</span>
              </Label>
              <div className="relative mt-1">
                <Input
                  id="client_secret"
                  type={showClientSecret ? 'text' : 'password'}
                  placeholder="客户端密钥（某些应用类型不需要）"
                  {...register('client_secret')}
                  className={`h-10 pr-10 ${
                    errors.client_secret
                      ? 'border-red-400 focus:border-red-500'
                      : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                  }`}
                  disabled={isSubmitting}
                />
                <button
                  type="button"
                  onClick={() => setShowClientSecret(!showClientSecret)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  {showClientSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.client_secret && (
                <p className="text-sm text-red-500 mt-1">{errors.client_secret.message}</p>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                公共客户端应用通常不需要客户端密钥
              </p>
            </div>

            <div>
              <Label htmlFor="refresh_token" className="text-gray-700 dark:text-gray-300">
                刷新令牌 <span className="text-red-500">*</span>
              </Label>
              <div className="relative mt-1">
                <Input
                  id="refresh_token"
                  type={showRefreshToken ? 'text' : 'password'}
                  placeholder="从OAuth2授权流程中获取的刷新令牌"
                  {...register('refresh_token')}
                  className={`h-10 pr-10 ${
                    errors.refresh_token
                      ? 'border-red-400 focus:border-red-500'
                      : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                  }`}
                  disabled={isSubmitting}
                />
                <button
                  type="button"
                  onClick={() => setShowRefreshToken(!showRefreshToken)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  {showRefreshToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.refresh_token && (
                <p className="text-sm text-red-500 mt-1">{errors.refresh_token.message}</p>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                用于获取访问令牌的长期有效令牌
              </p>
            </div>

            <div>
              <div className="flex items-center gap-2 mb-1">
                <Label htmlFor="scope" className="text-gray-700 dark:text-gray-300">
                  权限范围 <span className="text-gray-400">(可选)</span>
                </Label>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={fillDefaultScope}
                  className="text-xs text-blue-600 hover:text-blue-800 p-0 h-auto"
                >
                  使用默认
                </Button>
              </div>
              <Input
                id="scope"
                type="text"
                placeholder="权限范围，用空格分隔"
                {...register('scope')}
                className={`h-10 ${
                  errors.scope
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={isSubmitting}
              />
              {errors.scope && <p className="text-sm text-red-500 mt-1">{errors.scope.message}</p>}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                留空将使用默认权限：IMAP.AccessAsUser.All SMTP.Send offline_access
              </p>
            </div>
          </div>

          {/* 设置说明 */}
          <Collapsible open={showInstructions} onOpenChange={setShowInstructions}>
            <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
              {showInstructions ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
              详细配置说明
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-4">
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  Azure应用注册步骤：
                </h4>
                <ol className="text-sm text-gray-600 dark:text-gray-400 space-y-2 list-decimal list-inside">
                  <li>
                    访问{' '}
                    <a
                      href="https://portal.azure.com"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:text-blue-800 inline-flex items-center gap-1"
                    >
                      Azure Portal <ExternalLink className="w-3 h-3" />
                    </a>
                  </li>
                  <li>导航到"Azure Active Directory" → "应用注册"</li>
                  <li>点击"新注册"创建应用</li>
                  <li>设置重定向URI（可选）</li>
                  <li>在"概述"页面复制"应用程序（客户端）ID"</li>
                  <li>如需客户端密钥，在"证书和密码"中创建</li>
                  <li>在"API权限"中添加Microsoft Graph权限</li>
                </ol>

                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mt-4">
                  获取刷新令牌：
                </h4>
                <ol className="text-sm text-gray-600 dark:text-gray-400 space-y-2 list-decimal list-inside">
                  <li>使用OAuth2授权码流程获取初始令牌</li>
                  <li>确保请求中包含 offline_access 权限</li>
                  <li>从令牌响应中提取 refresh_token</li>
                  <li>刷新令牌可长期使用，无需重新授权</li>
                </ol>

                <div className="border-t border-gray-200 dark:border-gray-600 pt-3">
                  <h5 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                    注意事项：
                  </h5>
                  <ul className="text-xs text-gray-500 dark:text-gray-400 space-y-1 list-disc list-inside">
                    <li>公共客户端应用通常不需要客户端密钥</li>
                    <li>企业环境可能需要管理员同意权限</li>
                    <li>刷新令牌应安全存储，不要泄露</li>
                    <li>权限范围必须与应用注册时配置的一致</li>
                  </ul>
                </div>
              </div>
            </CollapsibleContent>
          </Collapsible>

          <div className="flex gap-3">
            <Button
              type="submit"
              disabled={isSubmitting}
              className="flex-1 h-10 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900"
            >
              {isSubmitting ? (
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                  配置中...
                </div>
              ) : (
                '创建账户'
              )}
            </Button>

            {onCancel && (
              <Button
                type="button"
                variant="outline"
                onClick={onCancel}
                disabled={isSubmitting}
                className="px-6 h-10 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300"
              >
                取消
              </Button>
            )}
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
