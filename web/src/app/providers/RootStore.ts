// src/app/providers/RootStore.ts
import { makeAutoObservable, runInAction } from 'mobx';
import type { IUser, IExchangeDeal, IChainLink } from '../../shared/api/types';
import { DealStatus } from '../../shared/api/types';
import { AuthStore } from '../hooks/stores/AuthStore';
import { dealStore } from './DealStore';

class RootStore {
  auth: AuthStore;
  deals = dealStore;
  currentUser: IUser | null = null;
  isLoading = false;

  constructor() {
    makeAutoObservable(this);
    this.auth = new AuthStore();
  }

  // Простой мок-логин без аргументов (для быстрой демонстрации)
  login = async () => {
    this.isLoading = true;
    try {
      await new Promise(resolve => setTimeout(resolve, 500));

      runInAction(() => {
        this.currentUser = {
          id: 1,
          username: 'Alex_Dev',
          rating: 4.8,
          declineCount: 1,
          pvzAddress: 'ПВЗ №123, ул. Ленина 10'
        };
        this.isLoading = false;
      });
    } catch {
      runInAction(() => { this.isLoading = false; });
    }
  };

  logout = () => {
    this.currentUser = null;
    this.auth.logout();
  };

  fetchCurrentUser = async () => {
    // Можно раскомментировать для авто-входа
    // this.login();
  };

  // Legacy method - delegate to dealStore
  updateDealStatus = (dealId: string, newStatus: DealStatus) => {
    if (newStatus === DealStatus.CANCELLED) {
      this.deals.cancelDeal(dealId, 'Сделка отменена пользователем');
    } else if (newStatus === DealStatus.ACTIVE || newStatus === DealStatus.CONFIRMED) {
      this.deals.confirmDeal(dealId);
    }
  };

  // Legacy method - delegate to dealStore
  setDeals = (deals: IExchangeDeal[]) => {
    runInAction(() => {
      this.deals.deals = deals;
    });
  };
}

export const rootStore = new RootStore();