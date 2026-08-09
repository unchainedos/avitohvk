// src/app/providers/ItemStore.ts
import { makeAutoObservable, runInAction } from 'mobx';
import type { IItem } from '../../shared/api/types';
import { getInitialItems } from '../../entities/item/api/itemApi';
// Backend integration notes:
// import { itemApi } from '../../entities/item/api/itemApi';
// When backend is ready, remove localStorage-only persistence from this store and fetch items from API:
//   const items = await itemApi.getMyItems(currentUserId);
//   runInAction(() => { this.items = items; });
// Then convert lock / transfer / revert methods to async API calls.

const STORAGE_KEY_ITEMS = 'items_state';

export class ItemStore {
  items: IItem[] = [];

  constructor() {
    makeAutoObservable(this);
    // Initialize items from API mock dataset (persisted state already applied)
    this.items = getInitialItems();

    // Listen for external changes (itemApi mutating initialMockItems) and reload
    if (typeof window !== 'undefined' && window.addEventListener) {
      window.addEventListener('items_state_changed', () => {
        runInAction(() => {
          this.items = getInitialItems();
          console.debug('ItemStore: reloaded items after items_state_changed event');
        });
      });
    }
  }

  // Return all items
  get all() {
    return this.items;
  }

  // Find item by id
  getById(id: number): IItem | undefined {
    return this.items.find(i => i.id === id);
  }

  // Lock multiple items by ids
  lockItems(itemIds: number[]): void {
    // Backend integration example:
    // async lockItems(itemIds: number[]): Promise<void> {
    //   await Promise.all(itemIds.map(id => itemApi.updateItem(id, { isLocked: true })));
    //   runInAction(() => {
    //     ...
    //   });
    // }
    runInAction(() => {
      itemIds.forEach(id => {
        const idx = this.items.findIndex(i => i.id === id);
        if (idx !== -1) {
          this.items[idx].isLocked = true;
          console.debug('ItemStore: lockItems ->', id);
        }
      });
      this.saveToStorage();
    });
  }

  // Unlock multiple items by ids
  unlockItems(itemIds: number[]): void {
    runInAction(() => {
      itemIds.forEach(id => {
        const idx = this.items.findIndex(i => i.id === id);
        if (idx !== -1) {
          this.items[idx].isLocked = false;
          console.debug('ItemStore: unlockItems ->', id);
        }
      });
      this.saveToStorage();
    });
  }

  // Transfer exclusive right (chown)
  transferRights(itemId: number, newHolderId: number, lockItem: boolean = false): void {
    // Backend integration example:
    // async transferRights(itemId: number, newHolderId: number, lockItem: boolean = false): Promise<void> {
    //   await itemApi.updateItem(itemId, { holderId: newHolderId, isLocked: lockItem });
    //   runInAction(() => {
    //     ...
    //   });
    // }
    runInAction(() => {
      const idx = this.items.findIndex(i => i.id === itemId);
      if (idx !== -1) {
        const prevHolder = this.items[idx].holderId;
        this.items[idx].holderId = newHolderId;
        if (lockItem) {
          this.items[idx].isLocked = true;
        }
        console.debug('ItemStore: transferRights ->', itemId, 'from', prevHolder, 'to', newHolderId, 'lock:', lockItem);
      }
      this.saveToStorage();
    });
  }

  // Revert rights: holderId -> authorId
  revertRights(itemId: number): void {
    // Backend integration example:
    // async revertRights(itemId: number): Promise<void> {
    //   const item = this.getById(itemId);
    //   if (!item) return;
    //   await itemApi.updateItem(itemId, { holderId: item.authorId, isLocked: false });
    //   runInAction(() => { ... });
    // }
    runInAction(() => {
      const idx = this.items.findIndex(i => i.id === itemId);
      if (idx !== -1) {
        this.items[idx].holderId = this.items[idx].authorId;
        this.items[idx].isLocked = false;
        console.debug('ItemStore: revertRights ->', itemId, 'reverted to author', this.items[idx].authorId);
      }
      this.saveToStorage();
    });
  }

  // Delete item by id
  deleteItem(itemId: number): void {
    runInAction(() => {
      const idx = this.items.findIndex(i => i.id === itemId);
      if (idx !== -1) {
        this.items.splice(idx, 1);
      }
      this.saveToStorage();
    });
  }

  // Reset all items to default ownership and unlocked state
  resetAll(): void {
    runInAction(() => {
      this.items.forEach(i => {
        i.isLocked = false;
        i.holderId = i.authorId;
      });
      this.saveToStorage();
    });
  }

  saveToStorage(): void {
    // LocalStorage persistence is a temporary client-side cache.
    // When the backend is ready, this method can remain as a UI cache,
    // or be removed once item updates are fully server-owned.
    try {
      const stateToSave = this.items.map(item => ({
        id: item.id,
        isLocked: item.isLocked,
        holderId: item.holderId,
      }));
      localStorage.setItem(STORAGE_KEY_ITEMS, JSON.stringify(stateToSave));
    } catch (e) {
      console.error('Failed to save items state from ItemStore:', e);
    }
  }
}

export const itemStore = new ItemStore();
