const DAY_IN_MS = 1000 * 60 * 60 * 24;

export function parseDateString(dateString: string): Date | null {
  if (!dateString) return null;

  const directDate = new Date(dateString);
  if (!Number.isNaN(directDate.getTime())) {
    return directDate;
  }

  const normalizedDate = dateString.includes('T') ? dateString : dateString.replace(' ', 'T');
  const normalizedWithTimezone = /([+-]\d{2}:\d{2}|Z)$/.test(normalizedDate)
    ? normalizedDate
    : `${normalizedDate}Z`;

  const fallbackDate = new Date(normalizedWithTimezone);
  if (!Number.isNaN(fallbackDate.getTime())) {
    return fallbackDate;
  }

  return null;
}

export function formatMobileEmailListTime(dateString: string): string {
  const date = parseDateString(dateString);
  if (!date) {
    return dateString;
  }

  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff <= 0) {
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  const days = Math.floor(diff / DAY_IN_MS);

  if (days === 0) {
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  if (days === 1) {
    return '昨天';
  }

  if (days < 7) {
    return `${days}天前`;
  }

  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
  });
}
