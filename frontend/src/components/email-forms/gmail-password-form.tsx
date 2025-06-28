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
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';

// 应用专用密码验证函数
const validateAppPassword = (password: string) => {
  // 移除空格
  const cleanPassword = password.replace(/\s/g, '');

  // 检查长度
  if (cleanPassword.length !== 16) {
    return '应用专用密码必须是16位字符';
  }

  // 检查字符类型（只允许字母和数字）
  if (!/^[a-zA-Z0-9]+$/.test(cleanPassword)) {
    return '应用专用密码只能包含字母和数字';
  }

  return true;
};

const gmailPasswordSchema = z.object({
  name: z.string().min(1, '请输入账户名称'),
  email: z
    .string()
    .email('请输入有效的Gmail地址')
    .refine(
      (email) => email.endsWith('@gmail.com') || email.endsWith('@googlemail.com'),
      '请输入Gmail邮箱地址'
    ),
  password: z.string().refine(validateAppPassword, {
    message: '请输入有效的16位应用专用密码',
  }),
});

type GmailPasswordForm = z.infer<typeof gmailPasswordSchema>;

interface GmailPasswordFormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function GmailPasswordForm({ onSuccess, onCancel }: GmailPasswordFormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showInstructions, setShowInstructions] = useState(false);
  const { addAccount } = useMailboxStore();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    watch,
  } = useForm<GmailPasswordForm>({
    resolver: zodResolver(gmailPasswordSchema),
  });

  // 监听密码输入，自动格式化显示
  const passwordValue = watch('password', '');

  const formatPassword = (value: string) => {
    // 移除所有空格
    const clean = value.replace(/\s/g, '');
    // 每4个字符添加一个空格
    return clean.replace(/(.{4})/g, '$1 ').trim();
  };

  const onSubmit = async (data: GmailPasswordForm) => {
    setIsSubmitting(true);
    try {
      // 移除密码中的空格
      const cleanPassword = data.password.replace(/\s/g, '');

      const response = await apiClient.createEmailAccount({
        name: data.name,
        email: data.email,
        provider: 'gmail',
        auth_method: 'password',
        password: cleanPassword,
      });

      if (response.success && response.data) {
        addAccount(response.data);
        toast.success('Gmail账户添加成功');
        reset();
        onSuccess?.();
      } else {
        throw new Error(response.message || '创建账户失败');
      }
    } catch (error: any) {
      console.error('Gmail password authentication failed:', error);
      toast.error(error.message || 'Gmail账户添加失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Gmail 应用专用密码
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          使用Google应用专用密码安全连接您的Gmail账户
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
                disabled={isSubmitting}
              />
              {errors.email && <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>}
            </div>

            <div>
              <Label htmlFor="password" className="text-gray-700 dark:text-gray-300">
                应用专用密码
              </Label>
              <div className="relative mt-1">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="xxxx xxxx xxxx xxxx"
                  {...register('password')}
                  className={`h-10 pr-10 ${
                    errors.password
                      ? 'border-red-400 focus:border-red-500'
                      : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                  }`}
                  disabled={isSubmitting}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.password && (
                <p className="text-sm text-red-500 mt-1">{errors.password.message}</p>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                16位字母和数字组合，可以用空格分隔
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
              如何获取应用专用密码？
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-3">
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">设置步骤：</h4>
                <ol className="text-sm text-gray-600 dark:text-gray-400 space-y-2 list-decimal list-inside">
                  <li>
                    访问{' '}
                    <a
                      href="https://myaccount.google.com"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:text-blue-800 inline-flex items-center gap-1"
                    >
                      Google 账户管理 <ExternalLink className="w-3 h-3" />
                    </a>
                  </li>
                  <li>点击左侧的"安全性"</li>
                  <li>确保已启用"两步验证"</li>
                  <li>在"登录 Google"部分，点击"应用专用密码"</li>
                  <li>选择"邮件"和您的设备类型</li>
                  <li>点击"生成"</li>
                  <li>复制生成的16位密码</li>
                </ol>

                <div className="border-t border-gray-200 dark:border-gray-600 pt-3">
                  <h5 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                    注意事项：
                  </h5>
                  <ul className="text-xs text-gray-500 dark:text-gray-400 space-y-1 list-disc list-inside">
                    <li>必须先启用两步验证</li>
                    <li>应用专用密码只能用于一个应用</li>
                    <li>不要使用您的Google账户密码</li>
                    <li>可以随时撤销不再使用的密码</li>
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
                  添加中...
                </div>
              ) : (
                '添加账户'
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
