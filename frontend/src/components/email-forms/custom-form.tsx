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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ChevronDown, ChevronRight, Eye, EyeOff, Server, Mail, Shield } from 'lucide-react';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';

// 配置模式类型
type ConfigMode = 'full' | 'imap-only' | 'smtp-only';

// 安全选项
const securityOptions = [
  { value: 'SSL', label: 'SSL/TLS', description: '加密连接（推荐）' },
  { value: 'STARTTLS', label: 'STARTTLS', description: '升级到加密连接' },
  { value: 'NONE', label: '无加密', description: '不推荐，仅测试使用' },
];

// 常用端口建议
const portSuggestions = {
  imap: {
    SSL: 993,
    STARTTLS: 143,
    NONE: 143,
  },
  smtp: {
    SSL: 465,
    STARTTLS: 587,
    NONE: 25,
  },
};

const customSchema = z
  .object({
    name: z.string().min(1, '请输入账户名称'),
    email: z.string().email('请输入有效的邮箱地址'),
    username: z.string().min(1, '请输入用户名'),
    password: z.string().min(1, '请输入密码'),
    config_mode: z.enum(['full', 'imap-only', 'smtp-only']),

    // IMAP配置（条件验证）
    imap_host: z.string().optional(),
    imap_port: z.number().min(1, '端口必须大于0').max(65535, '端口不能超过65535').optional(),
    imap_security: z.enum(['SSL', 'STARTTLS', 'NONE']).optional(),

    // SMTP配置（条件验证）
    smtp_host: z.string().optional(),
    smtp_port: z.number().min(1, '端口必须大于0').max(65535, '端口不能超过65535').optional(),
    smtp_security: z.enum(['SSL', 'STARTTLS', 'NONE']).optional(),
  })
  .refine(
    (data) => {
      // 根据配置模式验证必需字段
      if (data.config_mode === 'full' || data.config_mode === 'imap-only') {
        return data.imap_host && data.imap_port && data.imap_security;
      }
      return true;
    },
    {
      message: '请完整配置IMAP设置',
      path: ['imap_host'],
    }
  )
  .refine(
    (data) => {
      if (data.config_mode === 'full' || data.config_mode === 'smtp-only') {
        return data.smtp_host && data.smtp_port && data.smtp_security;
      }
      return true;
    },
    {
      message: '请完整配置SMTP设置',
      path: ['smtp_host'],
    }
  );

type CustomForm = z.infer<typeof customSchema>;

interface CustomFormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function CustomForm({ onSuccess, onCancel }: CustomFormProps) {
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
    setValue,
  } = useForm<CustomForm>({
    resolver: zodResolver(customSchema),
    defaultValues: {
      config_mode: 'full',
      imap_security: 'SSL',
      smtp_security: 'STARTTLS',
    },
  });

  const configMode = watch('config_mode');
  const imapSecurity = watch('imap_security');
  const smtpSecurity = watch('smtp_security');

  // 根据安全选项建议端口
  const suggestPort = (type: 'imap' | 'smtp', security: string) => {
    const port = portSuggestions[type][security as keyof typeof portSuggestions.imap];
    if (type === 'imap') {
      setValue('imap_port', port);
    } else {
      setValue('smtp_port', port);
    }
  };

  const onSubmit = async (data: CustomForm) => {
    setIsSubmitting(true);
    try {
      const requestData: any = {
        name: data.name,
        email: data.email,
        auth_method: 'password',
        username: data.username,
        password: data.password,
      };

      // 根据配置模式添加相应配置
      if (data.config_mode === 'full' || data.config_mode === 'imap-only') {
        requestData.imap_host = data.imap_host;
        requestData.imap_port = data.imap_port;
        requestData.imap_security = data.imap_security;
      }

      if (data.config_mode === 'full' || data.config_mode === 'smtp-only') {
        requestData.smtp_host = data.smtp_host;
        requestData.smtp_port = data.smtp_port;
        requestData.smtp_security = data.smtp_security;
      }

      const response = await apiClient.createCustomEmailAccount(requestData);

      if (response.success && response.data) {
        addAccount(response.data);
        toast.success('自定义邮箱账户添加成功');
        reset();
        onSuccess?.();
      } else {
        throw new Error(response.message || '创建账户失败');
      }
    } catch (error: any) {
      console.error('Custom email account creation failed:', error);
      toast.error(error.message || '自定义邮箱账户添加失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          自定义邮箱
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          配置自定义IMAP/SMTP服务器连接任何邮件服务商
        </p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* 基本信息 */}
          <div className="space-y-4">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
              <Mail className="w-4 h-4" />
              基本信息
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
                  邮箱地址
                </Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="your.email@example.com"
                  {...register('email')}
                  className={`mt-1 h-10 ${
                    errors.email
                      ? 'border-red-400 focus:border-red-500'
                      : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                  }`}
                  disabled={isSubmitting}
                />
                {errors.email && (
                  <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>
                )}
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Label htmlFor="username" className="text-gray-700 dark:text-gray-300">
                  用户名
                </Label>
                <Input
                  id="username"
                  type="text"
                  placeholder="通常与邮箱地址相同"
                  {...register('username')}
                  className={`mt-1 h-10 ${
                    errors.username
                      ? 'border-red-400 focus:border-red-500'
                      : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                  }`}
                  disabled={isSubmitting}
                />
                {errors.username && (
                  <p className="text-sm text-red-500 mt-1">{errors.username.message}</p>
                )}
              </div>

              <div>
                <Label htmlFor="password" className="text-gray-700 dark:text-gray-300">
                  密码
                </Label>
                <div className="relative mt-1">
                  <Input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    placeholder="邮箱密码或应用专用密码"
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
              </div>
            </div>
          </div>

          {/* 配置模式选择 */}
          <div className="space-y-4">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
              <Server className="w-4 h-4" />
              配置模式
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="radio"
                  value="full"
                  {...register('config_mode')}
                  className="text-gray-900"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">完整配置（收发件）</span>
              </label>
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="radio"
                  value="imap-only"
                  {...register('config_mode')}
                  className="text-gray-900"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">仅收件（IMAP）</span>
              </label>
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="radio"
                  value="smtp-only"
                  {...register('config_mode')}
                  className="text-gray-900"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">仅发件（SMTP）</span>
              </label>
            </div>
          </div>

          {/* IMAP配置 */}
          {(configMode === 'full' || configMode === 'imap-only') && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                IMAP配置（收件服务器）
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="md:col-span-2">
                  <Label htmlFor="imap_host" className="text-gray-700 dark:text-gray-300">
                    IMAP服务器地址
                  </Label>
                  <Input
                    id="imap_host"
                    type="text"
                    placeholder="imap.example.com"
                    {...register('imap_host')}
                    className={`mt-1 h-10 ${
                      errors.imap_host
                        ? 'border-red-400 focus:border-red-500'
                        : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                    }`}
                    disabled={isSubmitting}
                  />
                  {errors.imap_host && (
                    <p className="text-sm text-red-500 mt-1">{errors.imap_host.message}</p>
                  )}
                </div>

                <div>
                  <Label htmlFor="imap_port" className="text-gray-700 dark:text-gray-300">
                    端口
                  </Label>
                  <Input
                    id="imap_port"
                    type="number"
                    placeholder="993"
                    {...register('imap_port', { valueAsNumber: true })}
                    className={`mt-1 h-10 ${
                      errors.imap_port
                        ? 'border-red-400 focus:border-red-500'
                        : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                    }`}
                    disabled={isSubmitting}
                  />
                  {errors.imap_port && (
                    <p className="text-sm text-red-500 mt-1">{errors.imap_port.message}</p>
                  )}
                </div>
              </div>

              <div>
                <Label className="text-gray-700 dark:text-gray-300">安全选项</Label>
                <div className="mt-1 grid grid-cols-1 md:grid-cols-3 gap-3">
                  {securityOptions.map((option) => (
                    <label
                      key={option.value}
                      className="flex items-center space-x-2 cursor-pointer"
                    >
                      <input
                        type="radio"
                        value={option.value}
                        {...register('imap_security')}
                        onChange={() => suggestPort('imap', option.value)}
                        className="text-gray-900"
                      />
                      <div>
                        <span className="text-sm text-gray-700 dark:text-gray-300">
                          {option.label}
                        </span>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {option.description}
                        </p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* SMTP配置 */}
          {(configMode === 'full' || configMode === 'smtp-only') && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                SMTP配置（发件服务器）
              </h3>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="md:col-span-2">
                  <Label htmlFor="smtp_host" className="text-gray-700 dark:text-gray-300">
                    SMTP服务器地址
                  </Label>
                  <Input
                    id="smtp_host"
                    type="text"
                    placeholder="smtp.example.com"
                    {...register('smtp_host')}
                    className={`mt-1 h-10 ${
                      errors.smtp_host
                        ? 'border-red-400 focus:border-red-500'
                        : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                    }`}
                    disabled={isSubmitting}
                  />
                  {errors.smtp_host && (
                    <p className="text-sm text-red-500 mt-1">{errors.smtp_host.message}</p>
                  )}
                </div>

                <div>
                  <Label htmlFor="smtp_port" className="text-gray-700 dark:text-gray-300">
                    端口
                  </Label>
                  <Input
                    id="smtp_port"
                    type="number"
                    placeholder="587"
                    {...register('smtp_port', { valueAsNumber: true })}
                    className={`mt-1 h-10 ${
                      errors.smtp_port
                        ? 'border-red-400 focus:border-red-500'
                        : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                    }`}
                    disabled={isSubmitting}
                  />
                  {errors.smtp_port && (
                    <p className="text-sm text-red-500 mt-1">{errors.smtp_port.message}</p>
                  )}
                </div>
              </div>

              <div>
                <Label className="text-gray-700 dark:text-gray-300">安全选项</Label>
                <div className="mt-1 grid grid-cols-1 md:grid-cols-3 gap-3">
                  {securityOptions.map((option) => (
                    <label
                      key={option.value}
                      className="flex items-center space-x-2 cursor-pointer"
                    >
                      <input
                        type="radio"
                        value={option.value}
                        {...register('smtp_security')}
                        onChange={() => suggestPort('smtp', option.value)}
                        className="text-gray-900"
                      />
                      <div>
                        <span className="text-sm text-gray-700 dark:text-gray-300">
                          {option.label}
                        </span>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {option.description}
                        </p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* 配置说明 */}
          <Collapsible open={showInstructions} onOpenChange={setShowInstructions}>
            <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
              {showInstructions ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
              配置说明和常用设置
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-4">
                <div>
                  <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                    常用邮件服务商配置：
                  </h4>
                  <div className="text-xs text-gray-600 dark:text-gray-400 space-y-2">
                    <div>
                      <strong>Gmail:</strong> IMAP: imap.gmail.com:993(SSL), SMTP:
                      smtp.gmail.com:587(STARTTLS)
                    </div>
                    <div>
                      <strong>Outlook:</strong> IMAP: outlook.office365.com:993(SSL), SMTP:
                      smtp.office365.com:587(STARTTLS)
                    </div>
                    <div>
                      <strong>QQ邮箱:</strong> IMAP: imap.qq.com:993(SSL), SMTP:
                      smtp.qq.com:465(SSL)
                    </div>
                    <div>
                      <strong>163邮箱:</strong> IMAP: imap.163.com:993(SSL), SMTP:
                      smtp.163.com:465(SSL)
                    </div>
                  </div>
                </div>

                <div className="border-t border-gray-200 dark:border-gray-600 pt-3">
                  <h5 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                    注意事项：
                  </h5>
                  <ul className="text-xs text-gray-500 dark:text-gray-400 space-y-1 list-disc list-inside">
                    <li>建议使用SSL或STARTTLS加密连接</li>
                    <li>某些邮件服务商需要应用专用密码</li>
                    <li>企业邮箱请联系管理员获取配置信息</li>
                    <li>配置完成后系统会自动测试连接</li>
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
