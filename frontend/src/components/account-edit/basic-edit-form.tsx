'use client';

import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Eye, EyeOff, Save, TestTube } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import type { EmailAccount } from '@/types/email';
import type { AccountEditConfig } from './account-edit-config';

// 基础编辑表单验证schema
const basicEditSchema = z.object({
  name: z.string().min(1, '账户名称不能为空'),
  email: z.string().email('请输入有效的邮箱地址').optional(),
  password: z.string().optional(),
  is_active: z.boolean(),
});

type BasicEditForm = z.infer<typeof basicEditSchema>;

interface BasicEditFormProps {
  account: EmailAccount;
  config: AccountEditConfig;
  onSuccess: () => void;
  onCancel: () => void;
  updateAccount: (account: EmailAccount) => void;
}

export function BasicEditForm({
  account,
  config,
  onSuccess,
  onCancel,
  updateAccount,
}: BasicEditFormProps) {
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
  } = useForm<BasicEditForm>({
    resolver: zodResolver(basicEditSchema),
  });

  // 初始化表单数据
  useEffect(() => {
    if (account) {
      reset({
        name: account.name,
        email: account.email,
        password: '', // 密码字段留空
        is_active: account.is_active,
      });
    }
  }, [account, reset]);

  const onSubmit = async (data: BasicEditForm) => {
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
      if (config.editableFields.includes('password') && data.password && data.password.trim()) {
        updateData.password = data.password.trim();
      }
      if (config.editableFields.includes('is_active')) {
        updateData.is_active = data.is_active;
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
      {/* 账户信息 */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{config.title}</h3>
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

        {/* 密码/授权码 */}
        {config.showPassword && config.editableFields.includes('password') && (
          <div>
            <Label htmlFor="password">
              {config.providerType?.includes('gmail')
                ? '应用专用密码'
                : config.providerType?.includes('qq') || config.providerType?.includes('163')
                  ? '授权码'
                  : '密码'}
              （留空则不修改）
            </Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                {...register('password')}
                placeholder={
                  config.providerType?.includes('gmail')
                    ? '输入16位应用专用密码'
                    : config.providerType?.includes('qq') || config.providerType?.includes('163')
                      ? '输入16位授权码'
                      : '输入新密码或留空保持不变'
                }
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
