'use client';

import { UseFormReturn } from 'react-hook-form';
import { Globe, Info } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { PROXY_EXAMPLES } from '@/types/email';

interface ProxyConfigFieldsProps {
  form: UseFormReturn<any>;
  disabled?: boolean;
  compact?: boolean;
}

export function ProxyConfigFields({
  form,
  disabled = false,
  compact = false
}: ProxyConfigFieldsProps) {
  const { register, formState: { errors } } = form;

  return (
    <div className="space-y-4">
      {/* 标题 */}
      {!compact && (
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-blue-100 dark:bg-blue-900 rounded-lg">
            <Globe className="h-4 w-4 text-blue-600 dark:text-blue-400" />
          </div>
          <div>
            <h3 className="font-medium text-gray-900 dark:text-gray-100">
              代理设置
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              为此邮箱配置网络代理（可选）
            </p>
          </div>
        </div>
      )}

      {/* 代理URL输入 */}
      <div>
        <Label htmlFor="proxy_url" className="text-gray-700 dark:text-gray-300">
          代理URL（可选）
        </Label>
        <Input
          id="proxy_url"
          type="text"
          placeholder="http://proxy.company.com:8080"
          {...register('proxy_url')}
          className={`mt-1 h-10 ${
            errors.proxy_url
              ? 'border-red-400 focus:border-red-500'
              : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
          }`}
          disabled={disabled}
        />
        {errors.proxy_url && (
          <p className="text-sm text-red-500 mt-1">{String(errors.proxy_url.message || '')}</p>
        )}
      </div>

      {/* 代理说明和示例 */}
      {!compact && (
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            配置代理后，该邮箱的所有网络连接都将通过代理服务器进行。留空表示不使用代理。
          </AlertDescription>
        </Alert>
      )}

      {/* 代理URL示例 */}
      {!compact && (
        <div className="text-sm text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 p-3 rounded-lg">
          <p className="font-medium mb-2">支持的代理格式：</p>
          <div className="space-y-1">
            {PROXY_EXAMPLES.map((example, index) => (
              <div key={index}>
                <span className="font-medium">{example.label}：</span>
                <code className="ml-1 text-xs bg-gray-200 dark:bg-gray-600 px-1 rounded">
                  {example.example}
                </code>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
