import { Suspense } from 'react';
import { SearchPageClient } from '@/components/mailbox/search-page-client';
import { LoadingSkeleton } from '@/components/mailbox/loading-skeleton';
import { ProtectedRoute } from '@/components/auth/route-guard';

export default function SearchPage() {
  return (
    <ProtectedRoute>
      <Suspense fallback={<LoadingSkeleton />}>
        <SearchPageClient />
      </Suspense>
    </ProtectedRoute>
  );
}
