import { getErrorMessage } from './getErrorMessage';

export const isDialogCancelError = (error: unknown) => {
  const message = getErrorMessage(error).trim().toLowerCase();

  return message.includes('cancelled by user')
};
