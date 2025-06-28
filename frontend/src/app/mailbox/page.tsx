'use client';

import { DesktopOnlyRoute } from '@/components/auth/route-guard';
import { MailboxLayout } from '@/components/mailbox/mailbox-layout';

export default function MailboxPage() {
  return (
    <DesktopOnlyRoute>
      <MailboxLayout />
    </DesktopOnlyRoute>
  );
}
