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
import { ChevronDown, ChevronRight, ExternalLink } from 'lucide-react';
import { useOAuth2 } from '@/hooks/use-oauth';
import { toast } from 'sonner';

const outlookOAuth2Schema = z.object({
  name: z.string().min(1, '请输入账户名称'),
  email: z
    .string()
    .email('请输入有效的Outlook邮箱地址')
    .refine((email) => {
      const outlookDomains = ['@outlook.com', '@hotmail.com', '@live.com', '@msn.com'];
      return outlookDomains.some((domain) => email.endsWith(domain));
    }, '请输入Outlook相关域名的邮箱地址（@outlook.com, @hotmail.com, @live.com, @msn.com）'),
});

type OutlookOAuth2Form = z.infer<typeof outlookOAuth2Schema>;

interface OutlookOAuth2FormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function OutlookOAuth2Form({ onSuccess, onCancel }: OutlookOAuth2FormProps) {
  const [isAuthenticating, setIsAuthenticating] = useState(false);
  const [showInstructions, setShowInstructions] = useState(false);
  const { authenticateOutlook } = useOAuth2();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<OutlookOAuth2Form>({
    resolver: zodResolver(outlookOAuth2Schema),
  });

  const onSubmit = async (data: OutlookOAuth2Form) => {
    setIsAuthenticating(true);
    try {
      await authenticateOutlook(data.name, data.email);
      // 注意：由于使用直接跳转，这里的代码不会执行
      // 成功处理在OAuth回调页面中进行
    } catch (error: any) {
      // 错误已在authenticateOutlook中处理
      console.error('Outlook OAuth2 authentication failed:', error);
      setIsAuthenticating(false);
    }
  };

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Outlook OAuth2 认证
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          通过Microsoft官方授权安全连接您的Outlook账户
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
                disabled={isAuthenticating}
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
                disabled={isAuthenticating}
              />
              {errors.email && <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                支持 @outlook.com, @hotmail.com, @live.com, @msn.com
              </p>
            </div>
          </div>

          <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">授权流程</h4>
            <ol className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
              <li>1. 点击"开始授权"按钮</li>
              <li>2. 跳转到Microsoft授权页面</li>
              <li>3. 登录并授权花火邮箱访问Outlook</li>
              <li>4. 完成后自动返回并创建账户</li>
            </ol>
          </div>

          {/* 设置说明 */}
          <Collapsible open={showInstructions} onOpenChange={setShowInstructions}>
            <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
              {showInstructions ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
              Outlook OAuth2 详细说明
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-3">
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  支持的账户类型：
                </h4>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                  <li>
                    <strong>个人账户</strong>：@outlook.com, @hotmail.com, @live.com, @msn.com
                  </li>
                  <li>
                    <strong>企业账户</strong>：组织域名邮箱（需要管理员授权）
                  </li>
                </ul>

                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mt-4">
                  OAuth2 权限说明：
                </h4>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                  <li>
                    <strong>IMAP.AccessAsUser.All</strong>：访问邮箱内容
                  </li>
                  <li>
                    <strong>SMTP.Send</strong>：代表用户发送邮件
                  </li>
                  <li>
                    <strong>offline_access</strong>：获取刷新令牌
                  </li>
                </ul>

                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mt-4">
                  企业账户注意事项：
                </h4>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                  <li>可能需要管理员预先同意应用权限</li>
                  <li>某些组织可能禁用了第三方应用访问</li>
                  <li>如遇权限问题，请联系IT管理员</li>
                </ul>

                <div className="border-t border-gray-200 dark:border-gray-600 pt-3">
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    了解更多：
                    <a
                      href="https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-permissions-and-consent"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:text-blue-800 inline-flex items-center gap-1 ml-1"
                    >
                      Microsoft OAuth2 文档 <ExternalLink className="w-3 h-3" />
                    </a>
                  </p>
                </div>
              </div>
            </CollapsibleContent>
          </Collapsible>

          <div className="flex gap-3">
            <Button
              type="submit"
              disabled={isAuthenticating}
              className="flex-1 h-10 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900"
            >
              {isAuthenticating ? (
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                  授权中...
                </div>
              ) : (
                '开始授权'
              )}
            </Button>

            {onCancel && (
              <Button
                type="button"
                variant="outline"
                onClick={onCancel}
                disabled={isAuthenticating}
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
