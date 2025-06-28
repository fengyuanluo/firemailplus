/**
 * 邮件相关 Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient } from '@/lib/api';
import type { EmailAddress } from '@/types/email';
import { useMailboxStore } from '@/lib/store';

interface UseEmailsParams {
  account_id?: number;
  folder_id?: number;
  page?: number;
  page_size?: number;
  search?: string;
  is_read?: boolean;
  is_starred?: boolean;
}

export function useEmails(params?: UseEmailsParams) {
  const {
    emails,
    selectedEmail,
    selectedFolder,
    searchQuery,
    isLoading,
    setEmails,
    updateEmail,
    removeEmail,
    selectEmail,
    setLoading,
  } = useMailboxStore();

  const queryClient = useQueryClient();

  // 获取邮件列表
  const {
    data,
    isLoading: isQueryLoading,
    error,
  } = useQuery({
    queryKey: ['emails', params],
    queryFn: async () => {
      setLoading(true);
      try {
        const response = await apiClient.getEmails(params);
        if (response.success && response.data) {
          setEmails(response.data.emails);
          return response.data;
        }
        throw new Error(response.message || '获取邮件失败');
      } finally {
        setLoading(false);
      }
    },
    enabled: !!params?.account_id,
  });

  // 获取单个邮件详情
  const useEmailDetail = (id: number) => {
    return useQuery({
      queryKey: ['email', id],
      queryFn: async () => {
        const response = await apiClient.getEmail(id);
        if (response.success && response.data) {
          return response.data;
        }
        throw new Error(response.message || '获取邮件详情失败');
      },
      enabled: !!id,
    });
  };

  // 发送邮件
  const sendEmailMutation = useMutation({
    mutationFn: (email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
    }) => apiClient.sendEmail(email),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emails'] });
      toast.success('邮件发送成功');
    },
    onError: (error: any) => {
      toast.error(error.message || '邮件发送失败');
    },
  });

  // 标记为已读
  const markAsReadMutation = useMutation({
    mutationFn: (id: number) => apiClient.markEmailAsRead(id),
    onSuccess: (_, id) => {
      updateEmail(id, { is_read: true });
      toast.success('已标记为已读');
    },
    onError: (error: any) => {
      toast.error(error.message || '标记失败');
    },
  });

  // 标记为未读
  const markAsUnreadMutation = useMutation({
    mutationFn: (id: number) => apiClient.markEmailAsUnread(id),
    onSuccess: (_, id) => {
      updateEmail(id, { is_read: false });
      toast.success('已标记为未读');
    },
    onError: (error: any) => {
      toast.error(error.message || '标记失败');
    },
  });

  // 切换星标
  const toggleStarMutation = useMutation({
    mutationFn: (id: number) => apiClient.toggleEmailStar(id),
    onSuccess: (_, id) => {
      const email = emails.find((e) => e.id === id);
      if (email) {
        updateEmail(id, { is_starred: !email.is_starred });
        toast.success(email.is_starred ? '已取消星标' : '已添加星标');
      }
    },
    onError: (error: any) => {
      toast.error(error.message || '操作失败');
    },
  });

  // 删除邮件
  const deleteEmailMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteEmail(id),
    onSuccess: (_, id) => {
      removeEmail(id);
      if (selectedEmail?.id === id) {
        selectEmail(null);
      }
      toast.success('邮件删除成功');
    },
    onError: (error: any) => {
      toast.error(error.message || '删除失败');
    },
  });

  return {
    emails,
    selectedEmail,
    selectedFolder,
    searchQuery,
    isLoading: isLoading || isQueryLoading,
    error,
    selectEmail,
    sendEmail: sendEmailMutation.mutate,
    markAsRead: markAsReadMutation.mutate,
    markAsUnread: markAsUnreadMutation.mutate,
    toggleStar: toggleStarMutation.mutate,
    deleteEmail: deleteEmailMutation.mutate,
    isSending: sendEmailMutation.isPending,
    useEmailDetail,
  };
}
