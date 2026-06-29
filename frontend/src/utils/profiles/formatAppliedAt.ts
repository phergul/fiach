const parseAppliedTimestamp = (value: unknown): Date | null => {
  if (value === null || value === undefined) {
    return null;
  }

  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? null : value;
  }

  if (typeof value === 'string') {
    if (value.trim() === '') {
      return null;
    }

    const normalized = value.includes('T') ? value : `${value.replace(' ', 'T')}Z`;
    const date = new Date(normalized);
    return Number.isNaN(date.getTime()) ? null : date;
  }

  return null;
};

export const formatAppliedAt = (appliedAt: string) => {
  const date = parseAppliedTimestamp(appliedAt);
  if (date === null) {
    return 'Applied time unknown';
  }

  return formatAppliedAtFromDate(date);
};

export const formatAppliedAtFromDate = (appliedAt: Date | string | unknown) => {
  const date = parseAppliedTimestamp(appliedAt);
  if (date === null) {
    return 'Applied time unknown';
  }

  return `Applied ${new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)}`;
};
