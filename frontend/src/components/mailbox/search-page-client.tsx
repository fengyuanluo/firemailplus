'use client';

import { useIsMobile } from '@/hooks/use-responsive';
import { MailboxLayout } from './mailbox-layout';
import {
  SearchResultsPage,
  SearchResultsProvider,
  SearchResultsHeader,
  SearchResultsContent,
  SearchResultsMobileFilters,
} from './search-results-page';

export function SearchPageClient() {
  const isMobile = useIsMobile();

  if (isMobile) {
    return <SearchResultsPage />;
  }

  return (
    <SearchResultsProvider>
      <MailboxLayout header={<SearchResultsHeader />} showSidebar={false}>
        <SearchResultsContent />
      </MailboxLayout>
      <SearchResultsMobileFilters />
    </SearchResultsProvider>
  );
}
