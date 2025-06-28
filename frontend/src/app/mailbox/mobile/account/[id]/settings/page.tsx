'use client';

import { use } from 'react';
import { MobileAccountSettingsPage } from '@/components/mobile/mobile-account-settings-page';
import { MobileOnlyRoute } from '@/components/auth/route-guard';

interface PageProps {
  params: Promise<{
    id: string;
  }>;
}

export default function AccountSettingsPage({ params }: PageProps) {
  const resolvedParams = use(params);
  const accountId = parseInt(resolvedParams.id);

  return (
    <MobileOnlyRoute>
      <MobileAccountSettingsPage accountId={accountId} />
    </MobileOnlyRoute>
  );
}
