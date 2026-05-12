export const formatProfileEditedAt = (updatedAt: string) => {
  if (updatedAt.trim() === '') {
    return 'Edited time unknown';
  }

  const normalizedUpdatedAt = updatedAt.includes('T')
    ? updatedAt
    : `${updatedAt.replace(' ', 'T')}Z`;
  const date = new Date(normalizedUpdatedAt);
  if (Number.isNaN(date.getTime())) {
    return 'Edited time unknown';
  }

  return `Edited ${new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)}`;
};
