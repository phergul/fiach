const fallbackErrorMessage = 'Something went wrong.';
const genericFallbackErrorMessage = 'Something went wrong. Check the logs for details.';
const runtimeErrorMessage = 'RuntimeError';

const technicalErrorPattern =
  /\b(UNIQUE constraint failed|SQLITE|constraint failed|FOREIGN KEY constraint failed|\(\d{4}\))\b/i;

const sentenceCase = (message: string) => {
  const trimmedMessage = message.trim();
  if (trimmedMessage === '') {
    return trimmedMessage;
  }

  return `${trimmedMessage.charAt(0).toUpperCase()}${trimmedMessage.slice(1)}`;
};

const stripNoisyPrefix = (message: string) => {
  return message
    .replace(/^\s*(error|exception):\s*/i, '')
    .replace(/\s+/g, ' ')
    .trim();
};

const isWindowsDriveSeparator = (message: string, index: number) => {
  const driveLetterIndex = index - 1;
  const drivePrefixIndex = index - 2;

  return (
    driveLetterIndex >= 0 &&
    /^[a-z]$/i.test(message.charAt(driveLetterIndex)) &&
    (drivePrefixIndex < 0 || /[\s"'(]/.test(message.charAt(drivePrefixIndex))) &&
    (message.charAt(index + 1) === '\\' || message.charAt(index + 1) === '/')
  );
};

const splitErrorChain = (message: string) => {
  const segments: string[] = [];
  let currentSegment = '';

  for (let index = 0; index < message.length; index += 1) {
    const character = message.charAt(index);

    if (character === ':' && !isWindowsDriveSeparator(message, index)) {
      segments.push(currentSegment);
      currentSegment = '';
    } else {
      currentSegment = `${currentSegment}${character}`;
    }
  }

  segments.push(currentSegment);

  return segments
    .map(stripNoisyPrefix)
    .filter((segment) => segment !== '')
    .filter((segment, index, allSegments) => allSegments.indexOf(segment) === index);
};

const withTerminalPeriod = (message: string) => {
  const trimmedMessage = message.trim();
  if (trimmedMessage === '' || /[.!?]$/.test(trimmedMessage)) {
    return trimmedMessage;
  }

  return `${trimmedMessage}.`;
};

const friendlyFromRawMessage = (message: string) => {
  const normalizedMessage = stripNoisyPrefix(message);
  if (normalizedMessage === '') {
    return fallbackErrorMessage;
  }

  const segments = splitErrorChain(normalizedMessage);
  if (segments.length > 1 || technicalErrorPattern.test(normalizedMessage)) {
    return genericFallbackErrorMessage;
  }

  return withTerminalPeriod(sentenceCase(segments[0] ?? normalizedMessage));
};

const isObject = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null;
};

const isRuntimeEnvelopeMessage = (message: string) => {
  return stripNoisyPrefix(message) === runtimeErrorMessage;
};

const rawMessageFromObject = (error: Record<string, unknown>): string | null => {
  if (typeof error.error === 'string') {
    return error.error;
  }

  if (typeof error.cause === 'string') {
    return error.cause;
  }

  if (isObject(error.cause)) {
    const causeMessage = rawMessageFromObject(error.cause);
    if (causeMessage !== null) {
      return causeMessage;
    }
  }

  if (typeof error.message === 'string') {
    return error.message;
  }

  return null;
};

const rawMessageFromJSON = (message: string): string | null => {
  const trimmedMessage = message.trim();
  if (!trimmedMessage.startsWith('{')) {
    return null;
  }

  try {
    const parsedMessage: unknown = JSON.parse(trimmedMessage);
    if (!isObject(parsedMessage)) {
      return null;
    }

    const parsedRawMessage = rawMessageFromObject(parsedMessage);
    if (parsedRawMessage === null || isRuntimeEnvelopeMessage(parsedRawMessage)) {
      return null;
    }

    return parsedRawMessage;
  } catch {
    return null;
  }
};

const isSerializedRuntimeEnvelopeMessage = (message: string) => {
  const trimmedMessage = message.trim();
  if (!trimmedMessage.startsWith('{')) {
    return false;
  }

  try {
    const parsedMessage: unknown = JSON.parse(trimmedMessage);
    if (!isObject(parsedMessage)) {
      return false;
    }

    const parsedRawMessage = rawMessageFromObject(parsedMessage);

    return parsedRawMessage !== null && isRuntimeEnvelopeMessage(parsedRawMessage);
  } catch {
    return false;
  }
};

export const getRawErrorMessage = (error: unknown) => {
  if (error instanceof Error) {
    const errorCause = 'cause' in error ? error.cause : undefined;
    if (isObject(errorCause)) {
      const causeMessage = rawMessageFromObject(errorCause);
      if (causeMessage !== null && !isRuntimeEnvelopeMessage(causeMessage)) {
        return causeMessage;
      }
    }

    if (typeof errorCause === 'string' && !isRuntimeEnvelopeMessage(errorCause)) {
      return errorCause;
    }

    const jsonMessage = rawMessageFromJSON(error.message);
    if (jsonMessage !== null) {
      return jsonMessage;
    }

    if (
      isRuntimeEnvelopeMessage(error.message) ||
      isSerializedRuntimeEnvelopeMessage(error.message)
    ) {
      return fallbackErrorMessage;
    }

    return error.message;
  }

  if (typeof error === 'string') {
    const jsonMessage = rawMessageFromJSON(error);
    if (jsonMessage !== null) {
      return jsonMessage;
    }

    return isRuntimeEnvelopeMessage(error) || isSerializedRuntimeEnvelopeMessage(error)
      ? fallbackErrorMessage
      : error;
  }

  if (isObject(error)) {
    const objectMessage = rawMessageFromObject(error);
    if (objectMessage !== null && !isRuntimeEnvelopeMessage(objectMessage)) {
      return objectMessage;
    }

    return fallbackErrorMessage;
  }

  return fallbackErrorMessage;
};

export const getErrorMessage = (error: unknown) => {
  return friendlyFromRawMessage(getRawErrorMessage(error));
};
