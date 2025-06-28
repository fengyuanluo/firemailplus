import { Suspense } from 'react';
import { MobileOnlyRoute } from '@/components/auth/route-guard';
import { MobileAccountsPage } from '@/components/mobile/mobile-accounts-page';
import { MobileLoading } from '@/components/mobile/mobile-layout';

export default function MobileMailboxPage() {
  return (
    <MobileOnlyRoute>
      <Suspense fallback={<MobileLoading message="加载邮箱列表..." />}>
        <MobileAccountsPage />
      </Suspense>
    </MobileOnlyRoute>
  );
}
