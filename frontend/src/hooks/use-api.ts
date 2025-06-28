/**
 * 统一API调用Hook
 * 封装常用的API调用逻辑，消除重复代码
 */

import { useState, useCallback } from 'react';
import {
  useMutation,
  useQuery,
  useQueryClient,
  UseQueryOptions,
  UseMutationOptions,
} from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { handleError, ErrorType } from '@/lib/error-handler';
import { toast } from 'sonner';

// API响应类型
interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  message?: string;
  code?: string | number;
}

// API状态
interface ApiState<T = any> {
  data: T | null;
  isLoading: boolean;
  error: string | null;
  isSuccess: boolean;
  isError: boolean;
}

// API配置
interface ApiConfig {
  showSuccessToast?: boolean;
  showErrorToast?: boolean;
  successMessage?: string;
  errorMessage?: string;
  retryOnError?: boolean;
  maxRetries?: number;
}

const DEFAULT_CONFIG: ApiConfig = {
  showSuccessToast: false,
  showErrorToast: true,
  retryOnError: true,
  maxRetries: 3,
};

// 通用API Hook
export function useApi<T = any>(config: ApiConfig = {}) {
  const finalConfig = { ...DEFAULT_CONFIG, ...config };
  const [state, setState] = useState<ApiState<T>>({
    data: null,
    isLoading: false,
    error: null,
    isSuccess: false,
    isError: false,
  });

  const execute = useCallback(
    async (apiCall: () => Promise<ApiResponse<T>>): Promise<T | null> => {
      setState((prev) => ({ ...prev, isLoading: true, error: null, isError: false }));

      try {
        const response = await apiCall();

        if (response.success && response.data) {
          setState({
            data: response.data,
            isLoading: false,
            error: null,
            isSuccess: true,
            isError: false,
          });

          if (finalConfig.showSuccessToast && finalConfig.successMessage) {
            toast.success(finalConfig.successMessage);
          }

          return response.data;
        } else {
          throw new Error(response.message || '操作失败');
        }
      } catch (error: any) {
        const appError = handleError(error, 'api_call', {
          showToast: finalConfig.showErrorToast,
        });

        setState({
          data: null,
          isLoading: false,
          error: appError.message,
          isSuccess: false,
          isError: true,
        });

        return null;
      }
    },
    [finalConfig]
  );

  const reset = useCallback(() => {
    setState({
      data: null,
      isLoading: false,
      error: null,
      isSuccess: false,
      isError: false,
    });
  }, []);

  return {
    ...state,
    execute,
    reset,
  };
}

// 查询Hook
export function useApiQuery<T = any>(
  queryKey: string[],
  queryFn: () => Promise<ApiResponse<T>>,
  options: Partial<UseQueryOptions<T>> & ApiConfig = {}
) {
  const { showErrorToast = true, errorMessage, ...queryOptions } = options;

  return useQuery({
    queryKey,
    queryFn: async () => {
      try {
        const response = await queryFn();
        if (response.success && response.data) {
          return response.data;
        }
        throw new Error(response.message || '查询失败');
      } catch (error: any) {
        handleError(error, 'api_query', {
          showToast: showErrorToast,
        });
        throw error;
      }
    },
    retry: (failureCount, error: any) => {
      // 认证错误不重试
      if (error.status === 401) return false;
      return failureCount < 3;
    },
    staleTime: 5 * 60 * 1000, // 5分钟
    ...queryOptions,
  });
}

// 变更Hook
export function useApiMutation<TData = any, TVariables = any>(
  mutationFn: (variables: TVariables) => Promise<ApiResponse<TData>>,
  options: UseMutationOptions<TData, Error, TVariables> & ApiConfig = {}
) {
  const queryClient = useQueryClient();
  const {
    showSuccessToast = false,
    showErrorToast = true,
    successMessage,
    errorMessage,
    ...mutationOptions
  } = options;

  return useMutation({
    mutationFn: async (variables: TVariables) => {
      try {
        const response = await mutationFn(variables);
        if (response.success && response.data) {
          return response.data;
        }
        throw new Error(response.message || '操作失败');
      } catch (error: any) {
        handleError(error, 'api_mutation', {
          showToast: showErrorToast,
        });
        throw error;
      }
    },
    onSuccess: (data, variables, context) => {
      if (showSuccessToast && successMessage) {
        toast.success(successMessage);
      }
      mutationOptions.onSuccess?.(data, variables, context);
    },
    onError: (error, variables, context) => {
      // 错误已在mutationFn中处理
      mutationOptions.onError?.(error, variables, context);
    },
    ...mutationOptions,
  });
}

// 邮件相关API Hooks
export function useEmailsApi() {
  const queryClient = useQueryClient();

  // 获取邮件列表
  const getEmails = useApiQuery(['emails'], () => apiClient.getEmails(), {
    showErrorToast: true,
    errorMessage: '获取邮件列表失败',
  });

  // 发送邮件
  const sendEmail = useApiMutation((data: any) => apiClient.sendEmail(data), {
    showSuccessToast: true,
    successMessage: '邮件发送成功',
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emails'] });
    },
  });

  // 删除邮件
  const deleteEmail = useApiMutation((id: number) => apiClient.deleteEmail(id), {
    showSuccessToast: true,
    successMessage: '邮件删除成功',
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emails'] });
    },
  });

  return {
    getEmails,
    sendEmail,
    deleteEmail,
  };
}

// 账户相关API Hooks
export function useAccountsApi() {
  const queryClient = useQueryClient();

  // 获取账户列表
  const getAccounts = useApiQuery(['accounts'], () => apiClient.getEmailAccounts(), {
    showErrorToast: true,
    errorMessage: '获取账户列表失败',
  });

  // 添加账户
  const addAccount = useApiMutation((data: any) => apiClient.createEmailAccount(data), {
    showSuccessToast: true,
    successMessage: '账户添加成功',
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });

  // 删除账户
  const deleteAccount = useApiMutation((id: number) => apiClient.deleteEmailAccount(id), {
    showSuccessToast: true,
    successMessage: '账户删除成功',
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });

  return {
    getAccounts,
    addAccount,
    deleteAccount,
  };
}

// 通用CRUD Hook
export function useCrudApi<T = any>(
  resource: string,
  apiMethods: {
    getList: () => Promise<ApiResponse<T[]>>;
    getOne: (id: string | number) => Promise<ApiResponse<T>>;
    create: (data: Partial<T>) => Promise<ApiResponse<T>>;
    update: (id: string | number, data: Partial<T>) => Promise<ApiResponse<T>>;
    delete: (id: string | number) => Promise<ApiResponse<void>>;
  }
) {
  const queryClient = useQueryClient();

  const list = useApiQuery([resource, 'list'], apiMethods.getList, { showErrorToast: true });

  // getOne 方法需要在组件中单独使用 useApiQuery
  const getOne = useCallback(
    (id: string | number) => {
      // 返回查询配置，而不是直接调用Hook
      return {
        queryKey: [resource, 'detail', String(id)],
        queryFn: () => apiMethods.getOne(id),
        options: { showErrorToast: true },
      };
    },
    [resource, apiMethods.getOne]
  );

  const create = useApiMutation(apiMethods.create, {
    showSuccessToast: true,
    successMessage: '创建成功',
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [resource] });
    },
  });

  const update = useApiMutation(
    ({ id, data }: { id: string | number; data: Partial<T> }) => apiMethods.update(id, data),
    {
      showSuccessToast: true,
      successMessage: '更新成功',
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: [resource] });
      },
    }
  );

  const remove = useApiMutation(apiMethods.delete, {
    showSuccessToast: true,
    successMessage: '删除成功',
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [resource] });
    },
  });

  return {
    list,
    getOne,
    create,
    update,
    remove,
  };
}

// 批量操作Hook
export function useBatchApi<T = any>() {
  const [operations, setOperations] = useState<Array<() => Promise<any>>>([]);
  const [isExecuting, setIsExecuting] = useState(false);
  const [results, setResults] = useState<any[]>([]);

  const addOperation = useCallback((operation: () => Promise<any>) => {
    setOperations((prev) => [...prev, operation]);
  }, []);

  const execute = useCallback(async () => {
    if (operations.length === 0) return;

    setIsExecuting(true);
    const batchResults: any[] = [];

    try {
      for (const operation of operations) {
        const result = await operation();
        batchResults.push(result);
      }

      setResults(batchResults);
      toast.success(`批量操作完成，共处理 ${operations.length} 项`);
    } catch (error) {
      handleError(error, 'batch_operation');
    } finally {
      setIsExecuting(false);
      setOperations([]);
    }
  }, [operations]);

  const clear = useCallback(() => {
    setOperations([]);
    setResults([]);
  }, []);

  return {
    addOperation,
    execute,
    clear,
    isExecuting,
    operationCount: operations.length,
    results,
  };
}
