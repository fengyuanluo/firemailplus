/**
 * 批量添加邮箱账户的Hook
 */

import { useState, useCallback } from 'react';
import { apiClient } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';
import { toast } from 'sonner';

export interface BatchAccountData {
  email: string;
  password: string; // 在OAuth2中不使用，但保留用于格式解析
  client_id: string;
  refresh_token: string;
}

export interface BatchProcessResult {
  success: boolean;
  account?: any;
  error?: string;
  data: BatchAccountData;
}

export interface BatchProgress {
  total: number;
  processed: number;
  successful: number;
  failed: number;
  isProcessing: boolean;
  currentItem?: string;
  results: BatchProcessResult[];
}

// UUID格式验证
const isValidUUID = (value: string) => {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
  return uuidRegex.test(value);
};

// 邮箱格式验证
const isValidOutlookEmail = (email: string) => {
  const outlookDomains = ['@outlook.com', '@hotmail.com', '@live.com', '@msn.com'];
  return (
    outlookDomains.some((domain) => email.endsWith(domain)) &&
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
  );
};

// 解析批量数据
export const parseBatchData = (
  text: string
): {
  data: BatchAccountData[];
  errors: string[];
} => {
  const lines = text
    .trim()
    .split('\n')
    .filter((line) => line.trim());
  const data: BatchAccountData[] = [];
  const errors: string[] = [];
  const emailSet = new Set<string>();

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    const parts = line.trim().split('----');

    if (parts.length !== 4) {
      errors.push(`第${lineNumber}行：格式错误，应为"邮箱----密码----客户端ID----刷新令牌"`);
      return;
    }

    const [email, password, client_id, refresh_token] = parts.map((part) => part.trim());

    // 验证邮箱
    if (!isValidOutlookEmail(email)) {
      errors.push(`第${lineNumber}行：邮箱格式错误或不是Outlook域名`);
      return;
    }

    // 检查重复邮箱
    if (emailSet.has(email)) {
      errors.push(`第${lineNumber}行：邮箱地址重复`);
      return;
    }

    // 验证客户端ID
    if (!isValidUUID(client_id)) {
      errors.push(`第${lineNumber}行：客户端ID格式错误，应为UUID格式`);
      return;
    }

    // 验证刷新令牌
    if (!refresh_token || refresh_token.length < 10) {
      errors.push(`第${lineNumber}行：刷新令牌无效`);
      return;
    }

    emailSet.add(email);
    data.push({
      email,
      password,
      client_id,
      refresh_token,
    });
  });

  return { data, errors };
};

export function useBatchAddAccounts() {
  const [progress, setProgress] = useState<BatchProgress>({
    total: 0,
    processed: 0,
    successful: 0,
    failed: 0,
    isProcessing: false,
    results: [],
  });

  const { addAccount } = useMailboxStore();

  // 处理单个账户
  const processAccount = async (
    accountData: BatchAccountData,
    accountName: string
  ): Promise<BatchProcessResult> => {
    try {
      const response = await apiClient.createManualOAuth2Account({
        name: accountName,
        email: accountData.email,
        provider: 'outlook',
        client_id: accountData.client_id,
        refresh_token: accountData.refresh_token,
        scope:
          'https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/SMTP.Send offline_access',
      });

      if (response.success && response.data) {
        addAccount(response.data);
        return {
          success: true,
          account: response.data,
          data: accountData,
        };
      } else {
        return {
          success: false,
          error: response.message || '创建账户失败',
          data: accountData,
        };
      }
    } catch (error: any) {
      return {
        success: false,
        error: error.message || '网络错误',
        data: accountData,
      };
    }
  };

  // 批量处理账户
  const processBatch = useCallback(
    async (accounts: BatchAccountData[], namePrefix: string = 'Outlook账户') => {
      if (accounts.length === 0) {
        toast.error('没有有效的账户数据');
        return;
      }

      setProgress({
        total: accounts.length,
        processed: 0,
        successful: 0,
        failed: 0,
        isProcessing: true,
        results: [],
      });

      const results: BatchProcessResult[] = [];
      const batchSize = 5; // 并发控制，每批最多5个请求

      try {
        for (let i = 0; i < accounts.length; i += batchSize) {
          const batch = accounts.slice(i, i + batchSize);

          // 更新当前处理状态
          setProgress((prev) => ({
            ...prev,
            currentItem: `正在处理第 ${i + 1}-${Math.min(i + batchSize, accounts.length)} 个账户...`,
          }));

          // 并发处理当前批次
          const batchPromises = batch.map((account, batchIndex) => {
            const accountName = `${namePrefix} ${i + batchIndex + 1}`;
            return processAccount(account, accountName);
          });

          const batchResults = await Promise.allSettled(batchPromises);

          // 处理批次结果
          batchResults.forEach((result, batchIndex) => {
            let processResult: BatchProcessResult;

            if (result.status === 'fulfilled') {
              processResult = result.value;
            } else {
              processResult = {
                success: false,
                error: result.reason?.message || '处理失败',
                data: batch[batchIndex],
              };
            }

            results.push(processResult);

            // 更新进度
            setProgress((prev) => ({
              ...prev,
              processed: prev.processed + 1,
              successful: processResult.success ? prev.successful + 1 : prev.successful,
              failed: processResult.success ? prev.failed : prev.failed + 1,
              results: [...prev.results, processResult],
            }));
          });

          // 批次间延迟，避免服务器过载
          if (i + batchSize < accounts.length) {
            await new Promise((resolve) => setTimeout(resolve, 1000));
          }
        }

        // 完成处理
        setProgress((prev) => ({
          ...prev,
          isProcessing: false,
          currentItem: undefined,
        }));

        const successCount = results.filter((r) => r.success).length;
        const failCount = results.filter((r) => !r.success).length;

        if (successCount > 0) {
          toast.success(`批量添加完成：成功 ${successCount} 个，失败 ${failCount} 个`);
        } else {
          toast.error(`批量添加失败：所有 ${failCount} 个账户都添加失败`);
        }
      } catch (error: any) {
        setProgress((prev) => ({
          ...prev,
          isProcessing: false,
          currentItem: undefined,
        }));
        toast.error('批量处理过程中发生错误：' + error.message);
      }
    },
    [addAccount]
  );

  // 重试失败的账户
  const retryFailed = useCallback(
    async (namePrefix: string = 'Outlook账户') => {
      const failedAccounts = progress.results
        .filter((result) => !result.success)
        .map((result) => result.data);

      if (failedAccounts.length === 0) {
        toast.info('没有失败的账户需要重试');
        return;
      }

      await processBatch(failedAccounts, namePrefix);
    },
    [progress.results, processBatch]
  );

  // 重置进度
  const resetProgress = useCallback(() => {
    setProgress({
      total: 0,
      processed: 0,
      successful: 0,
      failed: 0,
      isProcessing: false,
      results: [],
    });
  }, []);

  return {
    progress,
    processBatch,
    retryFailed,
    resetProgress,
    parseBatchData,
  };
}
