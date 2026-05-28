import { getErrorMessage } from './getErrorMessage';

const dialogCancelMessages = [
  'operation was canceled',
  'operation was cancelled',
  'user canceled',
  'user cancelled',
  'dialog canceled',
  'dialog cancelled',
  'cancelled by the user',
  'canceled by the user',
];

export const isDialogCancelError = (error: unknown) => {
  const message = getErrorMessage(error).trim().toLowerCase();

  return dialogCancelMessages.some((cancelMessage) => message.includes(cancelMessage));
};
