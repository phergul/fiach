type CreateMapper<T = unknown> = (value: unknown) => T;

const identity = <T>(value: T): T => value;

export class CancellablePromise<T> extends Promise<T> {
  cancel() {}
}

export const Create = {
  Any: identity,
  ByteSlice: (value: unknown): string => (value === null || value === undefined ? '' : String(value)),
  Array:
    <T>(createItem: CreateMapper<T>) =>
    (value: unknown): T[] => {
      if (!Array.isArray(value)) {
        return [];
      }
      return value.map(createItem);
    },
  Events: {},
  Map:
    <TKey, TValue>(createKey: CreateMapper<TKey>, createValue: CreateMapper<TValue>) =>
    (value: unknown): Record<string, TValue> => {
      if (value !== null && typeof value === 'object') {
        return Object.fromEntries(
          Object.entries(value).map(([key, mapValue]) => [createKey(key) as string, createValue(mapValue)]),
        );
      }
      return {};
    },
  Nullable:
    <T>(createValue: CreateMapper<T>) =>
    (value: unknown): T | null =>
      value === null || value === undefined ? null : createValue(value),
  Struct:
    <T extends Record<string, CreateMapper>>(createField: T) =>
    (value: unknown): Record<string, unknown> => {
      if (value === null || typeof value !== 'object') {
        return {};
      }
      return Object.fromEntries(
        Object.entries(value).map(([key, fieldValue]) => [
          key,
          key in createField ? createField[key](fieldValue) : fieldValue,
        ]),
      );
    },
};

export const Call = {
  ByID: () => Promise.reject(new Error('Wails runtime calls must be mocked in tests.')),
};

export const Clipboard = {
  SetText: () => Promise.resolve(),
};

export const Dialogs = {
  OpenFile: () => Promise.resolve(''),
  SaveFile: () => Promise.resolve(''),
};

export const Events = {
  On: () => () => {},
};

export const Window = {
  Close: () => Promise.resolve(),
  IsMaximised: () => Promise.resolve(false),
  Maximise: () => Promise.resolve(),
  Minimise: () => Promise.resolve(),
  Restore: () => Promise.resolve(),
};

export const Application = {};
export const Browser = {};
export const Flags = {};
export const IOS = {};
export const Screens = {};
export const System = {};
export const WML = {};

export const clientId = 'vitest';
export const getTransport = () => undefined;
export const objectNames: string[] = [];
export const setTransport = () => {};
