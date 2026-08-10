// src/app/providers/DealStore.ts
import { makeAutoObservable, runInAction, reaction } from 'mobx';
import type { IExchangeDeal, IItem } from '../../shared/api/types';
import { DealStatus as DealStatusEnum, ChainLinkStatus, LogisticsStatus } from '../../shared/api/types';
import { itemStore } from './ItemStore';
import { mockUsers } from '../../entities/user/api/userApi';
// Backend integration notes:
// import { apiClient } from '../../shared/api/client';
// import { itemApi } from '../../entities/item/api/itemApi';
// When backend is ready, load deals from /v1/deals and keep local storage only as a fallback cache.

const STORAGE_KEY = 'exchange_app_deals';

export class DealStore {
  deals: IExchangeDeal[] = [];
  isLoading = false;

  constructor() {
    makeAutoObservable(this);
    this.loadFromStorage();

    // Auto-save on any changes to deals array
    reaction(
      () => this.deals.length + this.deals.map(d => d.status).join(','),
      () => this.saveToStorage(),
      { delay: 100 }
    );
  }

  // Load deals from localStorage
  // Backend integration note: replace this with an API call to /v1/deals when server persistence is available.
  loadFromStorage(): void {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsedDeals = JSON.parse(stored) as IExchangeDeal[];
        runInAction(() => {
          this.deals = parsedDeals;
          // Sync item lock states based on active deals
          this.syncItemLockStates();
        });
      }
    } catch (e) {
      console.error('Failed to load deals from storage:', e);
      localStorage.removeItem(STORAGE_KEY);
    }
  }

  // Save deals to localStorage
  saveToStorage(): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.deals));
    } catch (e) {
      console.error('Failed to save deals to storage:', e);
    }
  }

  // Sync item lock states based on deal statuses
  // Backend integration note: if the server tracks locks, this logic can be replaced
  // with direct item status updates from the backend and local cache refresh.
  syncItemLockStates(): void {
    // First, unlock all items via ItemStore
    itemStore.unlockItems(itemStore.all.map(i => i.id));

    // Then lock items only for active/confirmed deals
    this.deals.forEach(deal => {
      if (
        deal.status === DealStatusEnum.ACTIVE ||
        deal.status === DealStatusEnum.CONFIRMED
      ) {
        const idsToLock: number[] = [];
        deal.chain.forEach(link => {
          idsToLock.push(link.givingItem.id);
        });
        if (idsToLock.length > 0) {
          itemStore.lockItems(idsToLock);
        }
      }
    });
  }

  private findPendingDealForItemIds(itemIds: number[]): { deal: IExchangeDeal; insertAfterIndex: number } | undefined {
    for (const deal of this.deals) {
      if (deal.status !== DealStatusEnum.PENDING) continue;

      // Prefer matching currently held items (receivingItemId) before matching original giving items.
      for (const itemId of itemIds) {
        const receiveIndex = deal.chain.findIndex(link => link.receivingItemId === itemId);
        if (receiveIndex !== -1) {
          return { deal, insertAfterIndex: receiveIndex };
        }
      }

      for (const itemId of itemIds) {
        const giveIndex = deal.chain.findIndex(link => link.givingItemId === itemId);
        if (giveIndex !== -1) {
          return { deal, insertAfterIndex: giveIndex };
        }
      }
    }
    return undefined;
  }

  private appendChainLink(deal: IExchangeDeal, newLink: IExchangeDeal['chain'][number], insertAfterIndex: number) {
    if (insertAfterIndex < 0 || insertAfterIndex >= deal.chain.length) {
      deal.chain.push(newLink);
      return;
    }

    const insertIndex = insertAfterIndex + 1;
    const nextLink = deal.chain[insertIndex] || deal.chain[0];
    deal.chain.splice(insertIndex, 0, newLink);

    if (nextLink) {
      nextLink.receivingItemId = newLink.givingItemId;
      nextLink.receivingItem = newLink.givingItem;
      if (nextLink.status === ChainLinkStatus.ACCEPTED) {
        nextLink.status = ChainLinkStatus.PENDING;
      }
    }
  }

  // Create a new deal
  createDeal = async (
    initiatorId: number,
    targetItem: IItem,
    selectedGivingItems: IItem[],
    dealType: 'DIRECT' | 'CHAIN' = 'DIRECT'
  ): Promise<IExchangeDeal> => {
    // Backend integration example:
    // const response = await apiClient.post<IExchangeDeal>('/v1/deals', {
    //   initiatorId,
    //   targetItemId: targetItem.id,
    //   selectedGivingItemIds: selectedGivingItems.map(i => i.id),
    //   dealType,
    // });
    // const newDeal = response.data;
    // return newDeal;
    this.isLoading = true;

    await new Promise(r => setTimeout(r, 500));

    const givingItem = selectedGivingItems[0];
    const targetOwner = mockUsers[targetItem.holderId] || mockUsers[1];
    const givingIds = selectedGivingItems.map(i => i.id);

    const newChainLink = {
      userId: initiatorId,
      user: mockUsers[initiatorId] || mockUsers[1],
      status: ChainLinkStatus.ACCEPTED,
      givingItemId: givingItem.id,
      givingItem: givingItem,
      receivingItemId: targetItem.id,
      receivingItem: targetItem,
      logisticsStatus: LogisticsStatus.NONE,
    };

    let newDeal: IExchangeDeal;

    const existingDealMatch = this.findPendingDealForItemIds([targetItem.id, ...givingIds]);

    if (existingDealMatch) {
      runInAction(() => {
        this.appendChainLink(existingDealMatch.deal, newChainLink, existingDealMatch.insertAfterIndex);
        this.syncItemLockStates();
        this.saveToStorage();
        this.isLoading = false;
      });
      console.debug('DealStore: appendChainLink -> deal', existingDealMatch.deal.id, 'newLink:', newChainLink, 'insertAfterIndex:', existingDealMatch.insertAfterIndex);

      // When appending to an existing pending chain, transfer rights for the giving item to the target holder.
      itemStore.transferRights(givingItem.id, targetItem.holderId, false);
      return existingDealMatch.deal;
    }

    if (dealType === 'CHAIN') {
      const chain: IExchangeDeal['chain'] = [
        {
          ...newChainLink,
        },
        {
          userId: targetItem.holderId,
          user: targetOwner,
          status: ChainLinkStatus.PENDING,
          givingItemId: targetItem.id,
          givingItem: targetItem,
          receivingItemId: givingItem.id,
          receivingItem: givingItem,
          logisticsStatus: LogisticsStatus.NONE,
        },
      ];

      // Transfer rights for each giving item to the owner of targetItem, but do not lock until deal becomes active
      givingIds.forEach(id => itemStore.transferRights(id, targetItem.holderId, false));

      newDeal = {
        id: `deal-${Date.now()}`,
        status: DealStatusEnum.PENDING,
        deadline: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
        initiatorId,
        chain,
      };
    } else {
      const chain: IExchangeDeal['chain'] = [
        {
          ...newChainLink,
        },
        {
          userId: targetItem.holderId,
          user: targetOwner,
          status: ChainLinkStatus.PENDING,
          givingItemId: targetItem.id,
          givingItem: targetItem,
          receivingItemId: givingItem.id,
          receivingItem: givingItem,
          logisticsStatus: LogisticsStatus.NONE,
        },
      ];

      // Do not lock items until direct deal becomes active
      newDeal = {
        id: `deal-${Date.now()}`,
        status: DealStatusEnum.PENDING,
        deadline: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
        initiatorId,
        chain,
      };
    }

    console.debug('DealStore: createDeal ->', newDeal.id, 'initiator:', initiatorId, 'givingIds:', givingIds, 'targetItem:', targetItem.id, 'type:', dealType);

    runInAction(() => {
      this.deals.push(newDeal);
      this.syncItemLockStates();
      this.saveToStorage();
      this.isLoading = false;
    });

    return newDeal;
  };

  // Confirm a deal (for the recipient)
  confirmDeal = (dealId: string): void => {
    const deal = this.deals.find(d => d.id === dealId);
    if (!deal) return;

    // Backend integration example:
    // await apiClient.post(`/v1/deals/${dealId}/confirm`);

    runInAction(() => {
      deal.status = DealStatusEnum.ACTIVE;
      deal.chain.forEach(link => {
        link.status = ChainLinkStatus.ACCEPTED;
      });
      const itemIdsToLock = deal.chain.map(link => link.givingItem.id);
      itemStore.lockItems(itemIdsToLock);
      this.syncItemLockStates();
      this.saveToStorage();
    });
  };

  // Cancel/decline a deal
  cancelDeal = (dealId: string, reason?: string): void => {
    const deal = this.deals.find(d => d.id === dealId);
    if (!deal) return;

    // Backend integration example:
    // await apiClient.post(`/v1/deals/${dealId}/cancel`, { reason });

    runInAction(() => {
      deal.status = DealStatusEnum.CANCELLED;
      deal.declineReason = reason;

      // Unlock all items in the deal and revert rights back to authors
      deal.chain.forEach(link => {
        const givingId = link.givingItem.id;
        // Revert holderId back to author and unlock
        itemStore.revertRights(givingId);
        link.status = ChainLinkStatus.DECLINED;
      });

      this.syncItemLockStates();
      this.saveToStorage();
      // ItemStore already persisted items state
      console.debug('DealStore: cancelDeal ->', dealId, 'reason:', reason, 'deal:', deal);

    });
  };

  // Get pending deals for a specific user (Inbox)
  getPendingDealsForUser(userId: number): IExchangeDeal[] {
    return this.deals.filter(deal =>
      deal.status === DealStatusEnum.PENDING &&
      deal.chain.some(link => link.userId === userId && link.status === ChainLinkStatus.PENDING)
    );
  }

  // Get all deals involving a specific user
  getDealsForUser(userId: number): IExchangeDeal[] {
    return this.deals.filter(deal =>
      deal.chain.some(link => link.userId === userId)
    );
  }

  // Check whether item is reserved by any active/confirmed deal.
  // Pending deals do not block items and they can still be used in other chain proposals.
  isItemReserved(itemId: number): boolean {
    return this.deals.some(deal =>
      (deal.status === DealStatusEnum.ACTIVE || deal.status === DealStatusEnum.CONFIRMED) &&
      deal.chain.some(link => link.givingItemId === itemId || link.receivingItemId === itemId)
    );
  }

  // Get deal by ID
  getDealById(dealId: string): IExchangeDeal | undefined {
    return this.deals.find(d => d.id === dealId);
  }

  // Reset all deals and clear localStorage (for debug purposes)
  resetAllDeals = (): void => {
    runInAction(() => {
      // Clear localStorage for deals
      localStorage.removeItem(STORAGE_KEY);

      // Reset all items via ItemStore (revert holderId -> authorId and unlock)
      itemStore.resetAll();

      // Clear all deals
      this.deals = [];

      // ItemStore already persisted the items state; ensure deals storage is cleared
      localStorage.removeItem('items_state');

    });
  };
}

export const dealStore = new DealStore();