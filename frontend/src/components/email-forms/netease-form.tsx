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

// 客户端授权码验证函数
const validateAuthCode = (authCode: string) => {
  // 移除空格
  const cleanAuthCode = authCode.replace(/\s/g, '');

  // 检查长度
  if (cleanAuthCode.length !== 16) {
    return '客户端授权码必须是16位字符';
  }

  // 检查字符类型（只允许字母和数字）
  if (!/^[a-zA-Z0-9]+$/.test(cleanAuthCode)) {
    return '客户端授权码只能包含字母和数字';
  }

  return true;
};

const neteaseSchema = z.object({
  name: z.string().min(1, '请输入账户名称'),
  email: z
    .string()
    .email('请输入有效的网易邮箱地址')
    .refine((email) => {
      const neteaseDomains = [
        '@163.com',
        '@126.com',
        '@yeah.net',
        '@188.com',
        '@vip.163.com',
        '@vip.126.com',
      ];
      return neteaseDomains.some((domain) => email.endsWith(domain));
    }, '请输入网易相关域名的邮箱地址（@163.com, @126.com, @yeah.net, @188.com, @vip.163.com, @vip.126.com）'),
  password: z.string().refine(validateAuthCode, {
    message: '请输入有效的16位客户端授权码',
  }),
});

type NetEaseForm = z.infer<typeof neteaseSchema>;

interface NetEaseFormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function NetEaseForm({ onSuccess, onCancel }: NetEaseFormProps) {
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
  } = useForm<NetEaseForm>({
    resolver: zodResolver(neteaseSchema),
  });

  // 监听授权码输入，自动格式化显示
  const passwordValue = watch('password', '');

  const formatAuthCode = (value: string) => {
    // 移除所有空格
    const clean = value.replace(/\s/g, '');
    // 每4个字符添加一个空格
    return clean.replace(/(.{4})/g, '$1 ').trim();
  };

  const onSubmit = async (data: NetEaseForm) => {
    setIsSubmitting(true);
    try {
      // 移除授权码中的空格
      const cleanAuthCode = data.password.replace(/\s/g, '');

      const response = await apiClient.createEmailAccount({
        name: data.name,
        email: data.email,
        provider: '163',
        auth_method: 'password',
        password: cleanAuthCode,
      });

      if (response.success && response.data) {
        addAccount(response.data);
        toast.success('163邮箱账户添加成功');
        reset();
        onSuccess?.();
      } else {
        throw new Error(response.message || '创建账户失败');
      }
    } catch (error: any) {
      console.error('NetEase mail authentication failed:', error);
      toast.error(error.message || '163邮箱账户添加失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          网易邮箱
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          使用客户端授权码安全连接您的163/126/yeah邮箱账户
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
                网易邮箱地址
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="your.email@163.com"
                {...register('email')}
                className={`mt-1 h-10 ${
                  errors.email
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={isSubmitting}
              />
              {errors.email && <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                支持 @163.com, @126.com, @yeah.net, @188.com, @vip.163.com, @vip.126.com
              </p>
            </div>

            <div>
              <Label htmlFor="password" className="text-gray-700 dark:text-gray-300">
                客户端授权码
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
              如何获取网易邮箱客户端授权码？
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-3">
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">设置步骤：</h4>
                <ol className="text-sm text-gray-600 dark:text-gray-400 space-y-2 list-decimal list-inside">
                  <li>
                    访问{' '}
                    <a
                      href="https://mail.163.com"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:text-blue-800 inline-flex items-center gap-1"
                    >
                      网易邮箱网页版 <ExternalLink className="w-3 h-3" />
                    </a>
                  </li>
                  <li>点击"设置" → "POP3/SMTP/IMAP"</li>
                  <li>开启"IMAP/SMTP服务"</li>
                  <li>点击"客户端授权密码"</li>
                  <li>按照提示发送短信验证</li>
                  <li>获取16位客户端授权码</li>
                  <li>复制授权码到此处</li>
                </ol>

                <div className="border-t border-gray-200 dark:border-gray-600 pt-3">
                  <h5 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                    注意事项：
                  </h5>
                  <ul className="text-xs text-gray-500 dark:text-gray-400 space-y-1 list-disc list-inside">
                    <li>客户端授权码不是邮箱登录密码</li>
                    <li>授权码为16位字符，只包含字母和数字</li>
                    <li>如果忘记授权码，可以重新生成</li>
                    <li>每个授权码只能用于一个邮件客户端</li>
                    <li>126邮箱和yeah邮箱设置方法相同</li>
                  </ul>
                </div>

                <div className="border-t border-gray-200 dark:border-gray-600 pt-3">
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    了解更多：
                    <a
                      href="https://help.mail.163.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171312"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:text-blue-800 inline-flex items-center gap-1 ml-1"
                    >
                      网易邮箱客户端设置帮助 <ExternalLink className="w-3 h-3" />
                    </a>
                  </p>
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
