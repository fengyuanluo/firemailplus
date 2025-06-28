import { Suspense } from 'react';
import { MobileComposePage } from '@/components/mobile/mobile-compose-page';
import { MobileLoading } from '@/components/mobile/mobile-layout';
import { MobileOnlyRoute } from '@/components/auth/route-guard';

export default function MobileComposeRoute() {
  return (
    <MobileOnlyRoute>
      <Suspense fallback={<MobileLoading message="加载写信页面..." />}>
        <MobileComposePage />
      </Suspense>
    </MobileOnlyRoute>
  );
}
