import { Suspense } from 'react';
import { MobileEmailsPage } from '@/components/mobile/mobile-emails-page';
import { MobileLoading } from '@/components/mobile/mobile-layout';
import { MobileOnlyRoute } from '@/components/auth/route-guard';

interface MobileEmailsPageProps {
  params: Promise<{
    id: string;
  }>;
}

export default async function MobileFolderEmailsPage({ params }: MobileEmailsPageProps) {
  const { id } = await params;

  return (
    <MobileOnlyRoute>
      <Suspense fallback={<MobileLoading message="加载邮件列表..." />}>
        <MobileEmailsPage folderId={parseInt(id)} />
      </Suspense>
    </MobileOnlyRoute>
  );
}
