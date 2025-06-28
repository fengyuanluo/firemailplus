/**
 * 统一的移动端导航Hook
 * 简化移动端路由切换逻辑
 */

import { useRouter } from 'next/navigation';
import { useCallback } from 'react';

export function useMobileNavigation() {
  const router = useRouter();

  // 导航到搜索页面
  const navigateToSearch = useCallback(() => {
    router.push('/mailbox/search');
  }, [router]);

  // 导航到写信页面
  const navigateToCompose = useCallback(
    (params?: { reply?: string; replyAll?: string; forward?: string }) => {
      let url = '/mailbox/mobile/compose';

      if (params) {
        const searchParams = new URLSearchParams();
        if (params.reply) searchParams.set('reply', params.reply);
        if (params.replyAll) searchParams.set('replyAll', params.replyAll);
        if (params.forward) searchParams.set('forward', params.forward);

        if (searchParams.toString()) {
          url += `?${searchParams.toString()}`;
        }
      }

      router.push(url);
    },
    [router]
  );

  // 导航到邮件详情页面
  const navigateToEmailDetail = useCallback(
    (emailId: string | number) => {
      router.push(`/mailbox/mobile/email/${emailId}`);
    },
    [router]
  );

  // 导航到文件夹邮件列表
  const navigateToFolderEmails = useCallback(
    (accountId: string | number, folderId: string | number) => {
      router.push(`/mailbox/mobile/folder/${folderId}`);
    },
    [router]
  );

  // 导航到账户文件夹列表
  const navigateToAccountFolders = useCallback(
    (accountId: string | number) => {
      router.push(`/mailbox/mobile/account/${accountId}`);
    },
    [router]
  );

  // 导航到邮箱列表（主页）
  const navigateToMailboxHome = useCallback(() => {
    router.push('/mailbox/mobile');
  }, [router]);

  // 返回上一页
  const goBack = useCallback(() => {
    router.back();
  }, [router]);

  // 替换当前页面
  const replace = useCallback(
    (path: string) => {
      router.replace(path);
    },
    [router]
  );

  return {
    navigateToSearch,
    navigateToCompose,
    navigateToEmailDetail,
    navigateToFolderEmails,
    navigateToAccountFolders,
    navigateToMailboxHome,
    goBack,
    replace,
  };
}

// 移动端路由路径常量
export const MOBILE_ROUTES = {
  HOME: '/mailbox/mobile',
  SEARCH: '/mailbox/search',
  COMPOSE: '/mailbox/mobile/compose',
  ACCOUNT: (accountId: string | number) => `/mailbox/mobile/account/${accountId}`,
  FOLDER: (folderId: string | number) => `/mailbox/mobile/folder/${folderId}`,
  EMAIL: (emailId: string | number) => `/mailbox/mobile/email/${emailId}`,
} as const;

// 移动端路由工具函数
export const mobileRouteUtils = {
  // 检查是否为移动端路由
  isMobileRoute: (pathname: string) => {
    return pathname.includes('/mobile') || pathname.includes('/search');
  },

  // 获取对应的桌面端路由
  getDesktopRoute: (mobilePath: string) => {
    return mobilePath.replace('/mobile', '') || '/mailbox';
  },

  // 获取对应的移动端路由
  getMobileRoute: (desktopPath: string) => {
    if (desktopPath === '/mailbox') {
      return '/mailbox/mobile';
    }
    return desktopPath.replace('/mailbox', '/mailbox/mobile');
  },

  // 解析邮件ID从路径
  parseEmailId: (pathname: string) => {
    const match = pathname.match(/\/email\/(\d+)/);
    return match ? match[1] : null;
  },

  // 解析账户ID从路径
  parseAccountId: (pathname: string) => {
    const match = pathname.match(/\/account\/(\d+)/);
    return match ? match[1] : null;
  },

  // 解析文件夹ID从路径
  parseFolderId: (pathname: string) => {
    const match = pathname.match(/\/folder\/(\d+)/);
    return match ? match[1] : null;
  },
};
