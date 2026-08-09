// src/app/hooks/useAuthStore.ts
import { rootStore } from '../providers/RootStore';
import type { AuthStore } from './stores/AuthStore';

/**
 * Hook for accessing the AuthStore instance.
 * Should be used within components wrapped by observer from mobx-react-lite
 * to properly track reactive state changes.
 */
export const useAuthStore = (): AuthStore => {
  return rootStore.auth;
};