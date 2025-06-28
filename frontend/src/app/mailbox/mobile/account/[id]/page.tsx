import { Suspense } from 'react';
import { MobileFoldersPage } from '@/components/mobile/mobile-folders-page';
import { MobileLoading } from '@/components/mobile/mobile-layout';
import { MobileOnlyRoute } from '@/components/auth/route-guard';

interface MobileFoldersPageProps {
  params: Promise<{
    id: string;
  }>;
}

export default async function MobileAccountFoldersPage({ params }: MobileFoldersPageProps) {
  const { id } = await params;

  return (
    <MobileOnlyRoute>
      <Suspense fallback={<MobileLoading message="加载文件夹..." />}>
        <MobileFoldersPage accountId={parseInt(id)} />
      </Suspense>
    </MobileOnlyRoute>
  );
}
