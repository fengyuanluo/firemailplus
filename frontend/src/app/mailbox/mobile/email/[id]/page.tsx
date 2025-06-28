import { Suspense } from 'react';
import { MobileEmailDetailPage } from '@/components/mobile/mobile-email-detail-page';
import { MobileLoading } from '@/components/mobile/mobile-layout';
import { MobileOnlyRoute } from '@/components/auth/route-guard';

interface MobileEmailDetailPageProps {
  params: Promise<{
    id: string;
  }>;
}

export default async function MobileEmailDetailRoute({ params }: MobileEmailDetailPageProps) {
  const { id } = await params;

  return (
    <MobileOnlyRoute>
      <Suspense fallback={<MobileLoading message="加载邮件详情..." />}>
        <MobileEmailDetailPage emailId={parseInt(id)} />
      </Suspense>
    </MobileOnlyRoute>
  );
}
