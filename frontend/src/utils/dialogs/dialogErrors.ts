import { getErrorMessage } from '../errors';

export const isDialogCancelError = (error: unknown) => {
  const message = getErrorMessage(error).trim().toLowerCase();

  return message.includes('cancelled by user');
};
