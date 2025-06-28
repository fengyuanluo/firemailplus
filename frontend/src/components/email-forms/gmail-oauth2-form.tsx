'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useOAuth2 } from '@/hooks/use-oauth';
import { toast } from 'sonner';

const gmailOAuth2Schema = z.object({
  name: z.string().min(1, '请输入账户名称'),
  email: z
    .string()
    .email('请输入有效的Gmail地址')
    .refine((email) => email.endsWith('@gmail.com'), '请输入Gmail邮箱地址'),
});

type GmailOAuth2Form = z.infer<typeof gmailOAuth2Schema>;

interface GmailOAuth2FormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function GmailOAuth2Form({ onSuccess, onCancel }: GmailOAuth2FormProps) {
  const [isAuthenticating, setIsAuthenticating] = useState(false);
  const { authenticateGmail } = useOAuth2();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<GmailOAuth2Form>({
    resolver: zodResolver(gmailOAuth2Schema),
  });

  const onSubmit = async (data: GmailOAuth2Form) => {
    setIsAuthenticating(true);
    try {
      await authenticateGmail(data.name, data.email);
      toast.success('Gmail账户添加成功');
      reset();
      onSuccess?.();
    } catch (error: any) {
      // 错误已在authenticateGmail中处理
      console.error('Gmail OAuth2 authentication failed:', error);
    } finally {
      setIsAuthenticating(false);
    }
  };

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Gmail OAuth2 认证
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          通过Google官方授权安全连接您的Gmail账户
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
                Gmail地址
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="your.email@gmail.com"
                {...register('email')}
                className={`mt-1 h-10 ${
                  errors.email
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={isAuthenticating}
              />
              {errors.email && <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>}
            </div>
          </div>

          <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">授权流程</h4>
            <ol className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
              <li>1. 点击&quot;开始授权&quot;按钮</li>
              <li>2. 跳转到Google授权页面</li>
              <li>3. 登录并授权花火邮箱访问Gmail</li>
              <li>4. 完成后自动返回并创建账户</li>
            </ol>
          </div>

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
