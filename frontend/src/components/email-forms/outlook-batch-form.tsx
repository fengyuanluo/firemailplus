'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Progress } from '@/components/ui/progress';
import {
  ChevronDown,
  ChevronRight,
  Upload,
  Download,
  RefreshCw,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { useBatchAddAccounts, parseBatchData } from '@/hooks/use-batch-add';
import { toast } from 'sonner';

const outlookBatchSchema = z.object({
  batchData: z.string().min(1, '请输入批量数据'),
  namePrefix: z.string().min(1, '请输入账户名称前缀'),
});

type OutlookBatchForm = z.infer<typeof outlookBatchSchema>;

interface OutlookBatchFormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function OutlookBatchForm({ onSuccess, onCancel }: OutlookBatchFormProps) {
  const [showInstructions, setShowInstructions] = useState(false);
  const [showResults, setShowResults] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);

  const { progress, processBatch, retryFailed, resetProgress } = useBatchAddAccounts();

  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
    reset,
  } = useForm<OutlookBatchForm>({
    resolver: zodResolver(outlookBatchSchema),
    defaultValues: {
      namePrefix: 'Outlook账户',
    },
  });

  const batchDataValue = watch('batchData', '');

  // 实时验证批量数据
  const validateBatchData = (text: string) => {
    if (!text.trim()) {
      setValidationErrors([]);
      return;
    }

    const { errors } = parseBatchData(text);
    setValidationErrors(errors);
  };

  // 处理表单提交
  const onSubmit = async (data: OutlookBatchForm) => {
    const { data: accounts, errors } = parseBatchData(data.batchData);

    if (errors.length > 0) {
      toast.error(`数据格式错误：${errors.length} 个错误`);
      setValidationErrors(errors);
      return;
    }

    if (accounts.length === 0) {
      toast.error('没有有效的账户数据');
      return;
    }

    setShowResults(true);
    await processBatch(accounts, data.namePrefix);
  };

  // 导出结果
  const exportResults = () => {
    const results = progress.results.map((result) => ({
      邮箱: result.data.email,
      状态: result.success ? '成功' : '失败',
      错误信息: result.error || '',
    }));

    const csvContent = [
      '邮箱,状态,错误信息',
      ...results.map((row) => `${row.邮箱},${row.状态},${row.错误信息}`),
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `outlook_batch_results_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
  };

  // 重置表单
  const handleReset = () => {
    reset();
    resetProgress();
    setValidationErrors([]);
    setShowResults(false);
  };

  // 计算行数
  const lineCount = batchDataValue.split('\n').filter((line) => line.trim()).length;

  return (
    <Card className="border border-gray-200 dark:border-gray-700">
      <CardHeader>
        <CardTitle className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Outlook 批量添加
        </CardTitle>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          批量导入多个Outlook账户，支持OAuth2配置
        </p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="space-y-4">
            <div>
              <Label htmlFor="namePrefix" className="text-gray-700 dark:text-gray-300">
                账户名称前缀
              </Label>
              <input
                id="namePrefix"
                type="text"
                placeholder="Outlook账户"
                {...register('namePrefix')}
                className={`mt-1 h-10 w-full px-3 py-2 border rounded-md ${
                  errors.namePrefix
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                } bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100`}
                disabled={progress.isProcessing}
              />
              {errors.namePrefix && (
                <p className="text-sm text-red-500 mt-1">{errors.namePrefix.message}</p>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                账户将命名为：{watch('namePrefix', 'Outlook账户')} 1,{' '}
                {watch('namePrefix', 'Outlook账户')} 2...
              </p>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1">
                <Label htmlFor="batchData" className="text-gray-700 dark:text-gray-300">
                  批量数据
                </Label>
                <span className="text-xs text-gray-500 dark:text-gray-400">{lineCount} 行</span>
              </div>
              <Textarea
                id="batchData"
                placeholder="请输入批量数据，每行一个账户&#10;格式：邮箱----密码----客户端ID----刷新令牌&#10;&#10;示例：&#10;user1@outlook.com----password1----12345678-1234-1234-1234-123456789012----refresh_token_1&#10;user2@hotmail.com----password2----87654321-4321-4321-4321-210987654321----refresh_token_2"
                rows={8}
                {...register('batchData')}
                onChange={(e) => {
                  register('batchData').onChange(e);
                  validateBatchData(e.target.value);
                }}
                className={`${
                  errors.batchData || validationErrors.length > 0
                    ? 'border-red-400 focus:border-red-500'
                    : 'border-gray-300 dark:border-gray-600 focus:border-gray-900 dark:focus:border-gray-100'
                }`}
                disabled={progress.isProcessing}
              />
              {errors.batchData && (
                <p className="text-sm text-red-500 mt-1">{errors.batchData.message}</p>
              )}

              {/* 验证错误显示 */}
              {validationErrors.length > 0 && (
                <div className="mt-2 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <h4 className="text-sm font-medium text-red-800 dark:text-red-200 mb-2">
                    数据格式错误 ({validationErrors.length} 个)：
                  </h4>
                  <ul className="text-xs text-red-700 dark:text-red-300 space-y-1 max-h-32 overflow-y-auto">
                    {validationErrors.map((error, index) => (
                      <li key={index}>• {error}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </div>

          {/* 格式说明 */}
          <Collapsible open={showInstructions} onOpenChange={setShowInstructions}>
            <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
              {showInstructions ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
              数据格式说明
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-3">
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  数据格式要求：
                </h4>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-2 list-disc list-inside">
                  <li>
                    每行一个账户，格式：
                    <code className="bg-gray-200 dark:bg-gray-700 px-1 rounded">
                      邮箱----密码----客户端ID----刷新令牌
                    </code>
                  </li>
                  <li>
                    邮箱必须是Outlook相关域名（@outlook.com, @hotmail.com, @live.com, @msn.com）
                  </li>
                  <li>密码字段保留但不使用（OAuth2模式）</li>
                  <li>客户端ID必须是有效的UUID格式</li>
                  <li>刷新令牌必须是有效的OAuth2刷新令牌</li>
                </ul>

                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mt-4">
                  示例数据：
                </h4>
                <pre className="text-xs bg-gray-100 dark:bg-gray-700 p-2 rounded overflow-x-auto">
                  {`user1@outlook.com----password1----12345678-1234-1234-1234-123456789012----refresh_token_1
user2@hotmail.com----password2----87654321-4321-4321-4321-210987654321----refresh_token_2
user3@live.com----password3----11111111-2222-3333-4444-555555555555----refresh_token_3`}
                </pre>
              </div>
            </CollapsibleContent>
          </Collapsible>

          {/* 进度显示 */}
          {(progress.isProcessing || progress.total > 0) && (
            <div className="space-y-4">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    批量处理进度
                  </h4>
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    {progress.processed}/{progress.total}
                  </span>
                </div>

                <Progress
                  value={progress.total > 0 ? (progress.processed / progress.total) * 100 : 0}
                  className="mb-2"
                />

                <div className="flex justify-between text-xs text-gray-600 dark:text-gray-400">
                  <span>成功: {progress.successful}</span>
                  <span>失败: {progress.failed}</span>
                </div>

                {progress.currentItem && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    {progress.currentItem}
                  </p>
                )}
              </div>
            </div>
          )}

          {/* 结果显示 */}
          {showResults && progress.results.length > 0 && (
            <Collapsible open={showResults} onOpenChange={setShowResults}>
              <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
                {showResults ? (
                  <ChevronDown className="w-4 h-4" />
                ) : (
                  <ChevronRight className="w-4 h-4" />
                )}
                处理结果 ({progress.results.length} 个)
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-3">
                <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      处理结果
                    </h4>
                    <div className="flex gap-2">
                      {progress.failed > 0 && !progress.isProcessing && (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => retryFailed(watch('namePrefix'))}
                          className="text-xs"
                        >
                          <RefreshCw className="w-3 h-3 mr-1" />
                          重试失败
                        </Button>
                      )}
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={exportResults}
                        className="text-xs"
                      >
                        <Download className="w-3 h-3 mr-1" />
                        导出结果
                      </Button>
                    </div>
                  </div>

                  <div className="max-h-48 overflow-y-auto space-y-2">
                    {progress.results.map((result, index) => (
                      <div
                        key={index}
                        className={`flex items-center gap-2 p-2 rounded text-xs ${
                          result.success
                            ? 'bg-green-50 dark:bg-green-900/20 text-green-800 dark:text-green-200'
                            : 'bg-red-50 dark:bg-red-900/20 text-red-800 dark:text-red-200'
                        }`}
                      >
                        {result.success ? (
                          <CheckCircle className="w-3 h-3 flex-shrink-0" />
                        ) : (
                          <XCircle className="w-3 h-3 flex-shrink-0" />
                        )}
                        <span className="flex-1">{result.data.email}</span>
                        {result.error && <span className="text-xs opacity-75">{result.error}</span>}
                      </div>
                    ))}
                  </div>
                </div>
              </CollapsibleContent>
            </Collapsible>
          )}

          <div className="flex gap-3">
            <Button
              type="submit"
              disabled={progress.isProcessing || validationErrors.length > 0}
              className="flex-1 h-10 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900"
            >
              {progress.isProcessing ? (
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                  批量处理中...
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <Upload className="w-4 h-4" />
                  开始批量添加
                </div>
              )}
            </Button>

            <Button
              type="button"
              variant="outline"
              onClick={handleReset}
              disabled={progress.isProcessing}
              className="px-6 h-10 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300"
            >
              重置
            </Button>

            {onCancel && (
              <Button
                type="button"
                variant="outline"
                onClick={onCancel}
                disabled={progress.isProcessing}
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
