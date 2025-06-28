'use client';

import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Save, TestTube, RefreshCw, Eye, EyeOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { apiClient } from '@/lib/api';
import { useOAuth2 } from '@/hooks/use-oauth';
import { toast } from 'sonner';
import type { EmailAccount } from '@/types/email';
import type { AccountEditConfig } from './account-edit-config';

// OAuth2编辑表单验证schema
const oauth2EditSchema = z.object({
  name: z.string().min(1, '账户名称不能为空'),
  email: z.string().email('请输入有效的邮箱地址').optional(),
  client_id: z.string().optional(),
  client_secret: z.string().optional(),
  refresh_token: z.string().optional(),
  is_active: z.boolean(),
});

type OAuth2EditForm = z.infer<typeof oauth2EditSchema>;

interface OAuth2EditFormProps {
  account: EmailAccount;
  config: AccountEditConfig;
  onSuccess: () => void;
  onCancel: () => void;
  updateAccount: (account: EmailAccount) => void;
}

export function OAuth2EditForm({
  account,
  config,
  onSuccess,
  onCancel,
  updateAccount,
}: OAuth2EditFormProps) {
  const [showClientSecret, setShowClientSecret] = useState(false);
  const [showRefreshToken, setShowRefreshToken] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [isReauthing, setIsReauthing] = useState(false);
  const { authenticateGmail, authenticateOutlook } = useOAuth2();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    setValue,
    watch,
  } = useForm<OAuth2EditForm>({
    resolver: zodResolver(oauth2EditSchema),
  });

  // 初始化表单数据
  useEffect(() => {
    if (account) {
      reset({
        name: account.name,
        email: account.email,
        client_id: '', // OAuth2配置不显示敏感信息
        client_secret: '',
        refresh_token: '',
        is_active: account.is_active,
      });
    }
  }, [account, reset]);

  const onSubmit = async (data: OAuth2EditForm) => {
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
      if (config.editableFields.includes('is_active')) {
        updateData.is_active = data.is_active;
      }

      // OAuth2配置字段
      if (config.showOAuth2Config) {
        if (
          config.editableFields.includes('client_id') &&
          data.client_id &&
          data.client_id.trim()
        ) {
          updateData.client_id = data.client_id.trim();
        }
        if (
          config.editableFields.includes('client_secret') &&
          data.client_secret &&
          data.client_secret.trim()
        ) {
          updateData.client_secret = data.client_secret.trim();
        }
        if (
          config.editableFields.includes('refresh_token') &&
          data.refresh_token &&
          data.refresh_token.trim()
        ) {
          updateData.refresh_token = data.refresh_token.trim();
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

  const handleReauth = async () => {
    setIsReauthing(true);
    try {
      if (config.providerType === 'gmail-oauth2') {
        await authenticateGmail(account.name, account.email);
      } else if (config.providerType === 'outlook-oauth2') {
        await authenticateOutlook(account.name, account.email);
      }
      toast.success('重新授权成功');
      onSuccess();
    } catch (error: any) {
      console.error('Reauth failed:', error);
      toast.error(error.message || '重新授权失败');
    } finally {
      setIsReauthing(false);
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

        {/* OAuth2配置（仅手动配置类型显示） */}
        {config.showOAuth2Config && (
          <>
            <div>
              <Label htmlFor="client_id">客户端ID</Label>
              <Input
                id="client_id"
                {...register('client_id')}
                placeholder="输入新的客户端ID或留空保持不变"
              />
            </div>

            <div>
              <Label htmlFor="client_secret">客户端密钥（可选）</Label>
              <div className="relative">
                <Input
                  id="client_secret"
                  type={showClientSecret ? 'text' : 'password'}
                  {...register('client_secret')}
                  placeholder="输入新的客户端密钥或留空保持不变"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-2 top-1/2 -translate-y-1/2"
                  onClick={() => setShowClientSecret(!showClientSecret)}
                >
                  {showClientSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </Button>
              </div>
            </div>

            <div>
              <Label htmlFor="refresh_token">刷新令牌</Label>
              <div className="relative">
                <Input
                  id="refresh_token"
                  type={showRefreshToken ? 'text' : 'password'}
                  {...register('refresh_token')}
                  placeholder="输入新的刷新令牌或留空保持不变"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-2 top-1/2 -translate-y-1/2"
                  onClick={() => setShowRefreshToken(!showRefreshToken)}
                >
                  {showRefreshToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </Button>
              </div>
            </div>
          </>
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

        {/* 重新授权按钮 */}
        {config.showReauth && (
          <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <p className="text-sm text-blue-700 dark:text-blue-300 mb-3">
              如果遇到授权问题，可以重新进行OAuth2授权
            </p>
            <Button
              type="button"
              variant="outline"
              onClick={handleReauth}
              disabled={isReauthing}
              className="border-blue-300 text-blue-700 hover:bg-blue-50 dark:border-blue-600 dark:text-blue-300"
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isReauthing ? 'animate-spin' : ''}`} />
              {isReauthing ? '重新授权中...' : '重新授权'}
            </Button>
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
