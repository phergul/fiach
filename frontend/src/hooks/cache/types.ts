export type CachedKey = string | number;

export type CachePresence = 'hasKey' | 'hasValue';

export interface KeyedCachedResourceConfig<TData, TKey extends CachedKey> {
  emptyValue: TData;
  fetch: (key: TKey) => Promise<TData>;
  hasCachedEntry?: (key: TKey) => boolean;
  isEmpty?: (data: TData) => boolean;
  presence?: CachePresence;
}

export interface CachedResourceHandle<TData, TKey extends CachedKey> {
  clear: () => void;
  fetch: (key: TKey) => Promise<TData>;
  getCached: (key: TKey) => TData | undefined;
  invalidate: (key: TKey) => void;
  preload: (key: TKey) => Promise<TData>;
  setCached: (key: TKey, data: TData) => void;
  useCached: (key: TKey | null) => CachedHookResult<TData>;
}

export interface CachedHookResult<TData> {
  data: TData;
  isInitialLoading: boolean;
  isLoading: boolean;
  isRefreshing: boolean;
  loadError: string | null;
  refresh: () => Promise<TData | null>;
  setCached: (data: TData) => void;
}

export interface SingletonCachedResourceConfig<TData> {
  emptyValue: TData;
  fetch: () => Promise<TData>;
  hasCachedEntry?: () => boolean;
  isEmpty?: (data: TData) => boolean;
}

export interface SingletonCachedResourceHandle<TData> {
  clear: () => void;
  fetch: () => Promise<TData>;
  getCached: () => TData | undefined;
  invalidate: () => void;
  preload: () => Promise<TData>;
  setCached: (data: TData) => void;
  useCached: () => CachedHookResult<TData>;
}
