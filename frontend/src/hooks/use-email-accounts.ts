/**
 * 邮箱账户相关 Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient, type CreateAccountRequest } from '@/lib/api';
import { useMailboxStore } from '@/lib/store';

export function useEmailAccounts() {
  const { accounts, selectedAccount, setAccounts, addAccount, removeAccount, selectAccount } =
    useMailboxStore();
  const queryClient = useQueryClient();

  // 获取邮箱账户列表
  const { data, isLoading, error } = useQuery({
    queryKey: ['emailAccounts'],
    queryFn: async () => {
      const response = await apiClient.getEmailAccounts();
      if (response.success && response.data) {
        setAccounts(response.data);
        return response.data;
      }
      throw new Error(response.message || '获取邮箱账户失败');
    },
  });

  // 创建邮箱账户
  const createAccountMutation = useMutation({
    mutationFn: (account: CreateAccountRequest) => apiClient.createEmailAccount(account),
    onSuccess: (response) => {
      if (response.success && response.data) {
        addAccount(response.data);
        queryClient.invalidateQueries({ queryKey: ['emailAccounts'] });
        toast.success('邮箱账户创建成功');
      }
    },
    onError: (error: any) => {
      toast.error(error.message || '创建邮箱账户失败');
    },
  });

  // 删除邮箱账户
  const deleteAccountMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteEmailAccount(id),
    onSuccess: (_, id) => {
      removeAccount(id);
      queryClient.invalidateQueries({ queryKey: ['emailAccounts'] });
      toast.success('邮箱账户删除成功');
    },
    onError: (error: any) => {
      toast.error(error.message || '删除邮箱账户失败');
    },
  });

  return {
    accounts,
    selectedAccount,
    isLoading,
    error,
    selectAccount,
    createAccount: createAccountMutation.mutate,
    deleteAccount: deleteAccountMutation.mutate,
    isCreating: createAccountMutation.isPending,
    isDeleting: deleteAccountMutation.isPending,
  };
}
