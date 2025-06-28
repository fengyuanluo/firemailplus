'use client';

import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Eye, EyeOff, Save, TestTube, Server, Mail, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import type { EmailAccount } from '@/types/email';
import type { AccountEditConfig } from './account-edit-config';

// 自定义编辑表单验证schema
const customEditSchema = z.object({
  name: z.string().min(1, '账户名称不能为空'),
  email: z.string().email('请输入有效的邮箱地址').optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  imap_host: z.string().min(1, 'IMAP服务器不能为空'),
  imap_port: z.number().min(1).max(65535, '端口号必须在1-65535之间'),
  imap_security: z.enum(['SSL', 'STARTTLS', 'NONE']),
  smtp_host: z.string().min(1, 'SMTP服务器不能为空'),
  smtp_port: z.number().min(1).max(65535, '端口号必须在1-65535之间'),
  smtp_security: z.enum(['SSL', 'STARTTLS', 'NONE']),
  is_active: z.boolean(),
});

type CustomEditForm = z.infer<typeof customEditSchema>;

interface CustomEditFormProps {
  account: EmailAccount;
  config: AccountEditConfig;
  onSuccess: () => void;
  onCancel: () => void;
  updateAccount: (account: EmailAccount) => void;
}

export function CustomEditForm({
  account,
  config,
  onSuccess,
  onCancel,
  updateAccount,
}: CustomEditFormProps) {
  const [showPassword, setShowPassword] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTesting, setIsTesting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    setValue,
    watch,
  } = useForm<CustomEditForm>({
    resolver: zodResolver(customEditSchema),
  });

  // 初始化表单数据
  useEffect(() => {
    if (account) {
      reset({
        name: account.name,
        email: account.email,
        username: account.username || '',
        password: '', // 密码字段留空
        imap_host: account.imap_host,
        imap_port: account.imap_port,
        imap_security: account.imap_security as 'SSL' | 'STARTTLS' | 'NONE',
        smtp_host: account.smtp_host,
        smtp_port: account.smtp_port,
        smtp_security: account.smtp_security as 'SSL' | 'STARTTLS' | 'NONE',
        is_active: account.is_active,
      });
    }
  }, [account, reset]);

  const onSubmit = async (data: CustomEditForm) => {
    setIsSubmitting(true);
    try {
      const updateData: any = {};

      // 只更新可编辑的字段
      if (config.editableFields.includes('name')) {
        updateData.name = data.name;
      }
      if (config.editableFields.includes('email') && data.email) {
        updateData.email = data.email;
      }
      if (config.editableFields.includes('username') && data.username) {
        updateData.username = data.username;
      }
      if (config.editableFields.includes('password') && data.password && data.password.trim()) {
        updateData.password = data.password.trim();
      }
      if (config.editableFields.includes('is_active')) {
        updateData.is_active = data.is_active;
      }

      // IMAP/SMTP配置
      if (config.showImapSmtp) {
        if (config.editableFields.includes('imap_host')) {
          updateData.imap_host = data.imap_host;
        }
        if (config.editableFields.includes('imap_port')) {
          updateData.imap_port = data.imap_port;
        }
        if (config.editableFields.includes('imap_security')) {
          updateData.imap_security = data.imap_security;
        }
        if (config.editableFields.includes('smtp_host')) {
          updateData.smtp_host = data.smtp_host;
        }
        if (config.editableFields.includes('smtp_port')) {
          updateData.smtp_port = data.smtp_port;
        }
        if (config.editableFields.includes('smtp_security')) {
          updateData.smtp_security = data.smtp_security;
        }
      }

      const response = await apiClient.updateEmailAccount(account.id, updateData);

      if (response.success && response.data) {
        updateAccount(response.data);
        onSuccess();
      } else {
        throw new Error(response.message || '更新失败');
      }
    } catch (error: any) {
      console.error('Update account failed:', error);
      toast.error(error.message || '更新账户设置失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTestConnection = async () => {
    setIsTesting(true);
    try {
      const response = await apiClient.testEmailAccount(account.id);
      if (response.success) {
        toast.success('连接测试成功');
      } else {
        throw new Error(response.message || '连接测试失败');
      }
    } catch (error: any) {
      console.error('Test connection failed:', error);
      toast.error(error.message || '连接测试失败');
    } finally {
      setIsTesting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* 基本信息 */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Mail className="w-4 h-4" />
            {config.title}
          </h3>
          <span className="text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
            {config.providerType}
          </span>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400">{config.description}</p>

        {/* 账户名称 */}
        {config.editableFields.includes('name') && (
          <div>
            <Label htmlFor="name">账户名称</Label>
            <Input
              id="name"
              {...register('name')}
              className={errors.name ? 'border-red-500' : ''}
            />
            {errors.name && <p className="text-sm text-red-500 mt-1">{errors.name.message}</p>}
          </div>
        )}

        {/* 邮箱地址 */}
        {config.editableFields.includes('email') ? (
          <div>
            <Label htmlFor="email">邮箱地址</Label>
            <Input
              id="email"
              type="email"
              {...register('email')}
              className={errors.email ? 'border-red-500' : ''}
            />
            {errors.email && <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>}
          </div>
        ) : (
          <div>
            <Label>邮箱地址</Label>
            <div className="text-sm text-gray-600 dark:text-gray-400 p-2 bg-gray-50 dark:bg-gray-700 rounded">
              {account.email}
            </div>
          </div>
        )}

        {/* 用户名 */}
        {config.editableFields.includes('username') && (
          <div>
            <Label htmlFor="username">用户名</Label>
            <Input id="username" {...register('username')} />
          </div>
        )}

        {/* 密码 */}
        {config.showPassword && config.editableFields.includes('password') && (
          <div>
            <Label htmlFor="password">密码（留空则不修改）</Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                {...register('password')}
                placeholder="输入新密码或留空保持不变"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-2 top-1/2 -translate-y-1/2"
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </Button>
            </div>
          </div>
        )}

        {/* 启用状态 */}
        {config.editableFields.includes('is_active') && (
          <div className="flex items-center space-x-2">
            <Switch
              id="is_active"
              checked={watch('is_active')}
              onCheckedChange={(checked) => setValue('is_active', checked)}
            />
            <Label htmlFor="is_active">启用此账户</Label>
          </div>
        )}
      </div>

      {/* IMAP设置 */}
      {config.showImapSmtp && (
        <div className="space-y-4">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Server className="w-4 h-4" />
            IMAP设置
          </h3>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="imap_host">IMAP服务器</Label>
              <Input
                id="imap_host"
                {...register('imap_host')}
                className={errors.imap_host ? 'border-red-500' : ''}
              />
              {errors.imap_host && (
                <p className="text-sm text-red-500 mt-1">{errors.imap_host.message}</p>
              )}
            </div>

            <div>
              <Label htmlFor="imap_port">IMAP端口</Label>
              <Input
                id="imap_port"
                type="number"
                {...register('imap_port', { valueAsNumber: true })}
                className={errors.imap_port ? 'border-red-500' : ''}
              />
              {errors.imap_port && (
                <p className="text-sm text-red-500 mt-1">{errors.imap_port.message}</p>
              )}
            </div>
          </div>

          <div>
            <Label htmlFor="imap_security">IMAP安全类型</Label>
            <Select
              value={watch('imap_security')}
              onValueChange={(value) =>
                setValue('imap_security', value as 'SSL' | 'STARTTLS' | 'NONE')
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="SSL">SSL/TLS</SelectItem>
                <SelectItem value="STARTTLS">STARTTLS</SelectItem>
                <SelectItem value="NONE">无加密</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      )}

      {/* SMTP设置 */}
      {config.showImapSmtp && (
        <div className="space-y-4">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Shield className="w-4 h-4" />
            SMTP设置
          </h3>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="smtp_host">SMTP服务器</Label>
              <Input
                id="smtp_host"
                {...register('smtp_host')}
                className={errors.smtp_host ? 'border-red-500' : ''}
              />
              {errors.smtp_host && (
                <p className="text-sm text-red-500 mt-1">{errors.smtp_host.message}</p>
              )}
            </div>

            <div>
              <Label htmlFor="smtp_port">SMTP端口</Label>
              <Input
                id="smtp_port"
                type="number"
                {...register('smtp_port', { valueAsNumber: true })}
                className={errors.smtp_port ? 'border-red-500' : ''}
              />
              {errors.smtp_port && (
                <p className="text-sm text-red-500 mt-1">{errors.smtp_port.message}</p>
              )}
            </div>
          </div>

          <div>
            <Label htmlFor="smtp_security">SMTP安全类型</Label>
            <Select
              value={watch('smtp_security')}
              onValueChange={(value) =>
                setValue('smtp_security', value as 'SSL' | 'STARTTLS' | 'NONE')
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="SSL">SSL/TLS</SelectItem>
                <SelectItem value="STARTTLS">STARTTLS</SelectItem>
                <SelectItem value="NONE">无加密</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      )}

      {/* 底部按钮 */}
      <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
        <Button type="button" variant="outline" onClick={handleTestConnection} disabled={isTesting}>
          <TestTube className="w-4 h-4 mr-2" />
          {isTesting ? '测试中...' : '测试连接'}
        </Button>

        <div className="flex gap-3">
          <Button type="button" variant="outline" onClick={onCancel}>
            取消
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            <Save className="w-4 h-4 mr-2" />
            {isSubmitting ? '保存中...' : '保存'}
          </Button>
        </div>
      </div>
    </form>
  );
}
