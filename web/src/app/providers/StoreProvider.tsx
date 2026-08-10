// src/app/providers/StoreProvider.tsx
import React from 'react';
import { rootStore } from './RootStore';
import type { AuthStore } from '../hooks/stores/AuthStore';

const StoreContext = React.createContext(rootStore);

export const StoreProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <StoreContext.Provider value={rootStore}>
      {children}
    </StoreContext.Provider>
  );
};

export const useStore = () => React.useContext(StoreContext);

export const useAuthStore = (): AuthStore => {
  return rootStore.auth;
};