import { Suspense } from 'react';
import { SearchResultsPage } from '@/components/mailbox/search-results-page';
import { LoadingSkeleton } from '@/components/mailbox/loading-skeleton';
import { ProtectedRoute } from '@/components/auth/route-guard';

export default function SearchPage() {
  return (
    <ProtectedRoute>
      <Suspense fallback={<LoadingSkeleton />}>
        <SearchResultsPage />
      </Suspense>
    </ProtectedRoute>
  );
}
