// src/entities/item/api/itemApi.ts
import type { IItem, IExchangeDeal, IChainLink } from '../../../shared/api/types';
import { DealStatus, ChainLinkStatus, LogisticsStatus } from '../../../shared/api/types';
import { mockUsers } from '../../user/api/userApi';
// import { apiClient } from '../../../shared/api/client';

const STORAGE_KEY_ITEMS = 'items_state';

// Helper function to load items state from localStorage
const loadItemsState = (): Partial<IItem>[] => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY_ITEMS);
    if (stored) {
      return JSON.parse(stored);
    }
  } catch (e) {
    console.error('Failed to load items state:', e);
  }
  return [];
};

// Helper function to save items state to localStorage
const saveItemsState = (items: IItem[]): void => {
  try {
    // Only save mutable fields (isLocked, holderId) for each item
    const stateToSave = items.map(item => ({
      id: item.id,
      isLocked: item.isLocked,
      holderId: item.holderId,
    }));
    localStorage.setItem(STORAGE_KEY_ITEMS, JSON.stringify(stateToSave));
  } catch (e) {
    console.error('Failed to save items state:', e);
  }
};

// Apply persisted state to items
const applyPersistedState = (items: IItem[]): IItem[] => {
  const persistedState = loadItemsState();
  if (persistedState.length === 0) return items;

  return items.map(item => {
    const state = persistedState.find(s => s.id === item.id);
    if (state) {
      return {
        ...item,
        isLocked: state.isLocked ?? item.isLocked,
        holderId: state.holderId ?? item.holderId,
      };
    }
    return item;
  });
};

// Items organized by user (authorId)
// Each user has their own unique items with specific wishes
const initialMockItems: IItem[] = applyPersistedState([
  // === Alex_Dev (id: 1) items ===
  {
    id: 102,
    title: 'Игровая приставка PS5',
    description: 'Полный комплект, 2 геймпада, 3 диска в подарок. Без царапин.',
    imageUrl: 'https://images.unsplash.com/photo-1606144042614-b2417e99c4e3?w=800&q=80',
    images: [
        'https://images.unsplash.com/photo-1606144042614-b2417e99c4e3?w=800&q=80',
        'https://images.unsplash.com/photo-1607853202273-797f1c22a38e?w=800&q=80'
    ],
    category: 'Электроника',
    quantity: 1,
    unit: 'шт',
    authorId: 1,
    holderId: 1,
    isLocked: false,
    createdAt: '2026-08-02T12:00:00Z',
    wishes: ['Смартфон', 'Наушники'],
  },
  {
    id: 107,
    title: 'Кофемашина',
    description: 'Автоматическая, делает капучино. Требует чистки от накипи.',
    imageUrl: 'https://images.unsplash.com/photo-1517668808822-9ebb02f2a0e6?w=800&q=80',
    images: ['https://images.unsplash.com/photo-1517668808822-9ebb02f2a0e6?w=800&q=80'],
    category: 'Бытовая техника',
    quantity: 1,
    unit: 'шт',
    authorId: 1, // Alex's coffee machine
    holderId: 1,
    isLocked: false,
    createdAt: '2026-08-05T08:00:00Z',
    wishes: ['Книги', 'Винил'],
  },

  // === Dima_Biker (id: 2) items ===
  {
    id: 101,
    title: 'Велосипед горный',
    description: 'Хорошее состояние, 21 скорость. Идеален для города. Торг уместен при быстром обмене.',
    imageUrl: 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT_z4VgQKopNA_8vUS1LOGJ_UFbohFi7gYI_TuzfNMtlYQW4GaRnZn9xf4&s=10',
    images: [
        'https://pro-bike.ru/data/images/posts/32/40932/am9a8836-edit-285f332.jpg',
        'https://images.unsplash.com/photo-1576435728678-38d01d12e3b5?w=800&q=80',
        'https://images.unsplash.com/photo-1511994298220-412704691162?w=800&q=80'
    ],
    category: 'Спорт',
    quantity: 1,
    unit: 'шт',
    authorId: 2,
    holderId: 2, // Dima owns and holds his bicycle
    isLocked: false,
    createdAt: '2026-08-01T10:00:00Z',
    wishes: ['Апельсины', 'Лодка', 'Гитара'],
  },
  {
    id: 201,
    title: 'Мотоциклетный шлем',
    description: 'Полная защита, размер L. Новый, в коробке.',
    imageUrl: 'https://images.unsplash.com/photo-1591635566279-7838f5f075ae?w=800&q=80',
    images: ['https://images.unsplash.com/photo-1591635566279-7838f5f075ae?w=800&q=80'],
    category: 'Мототехника',
    quantity: 1,
    unit: 'шт',
    authorId: 2,
    holderId: 2,
    isLocked: false,
    createdAt: '2026-08-03T14:00:00Z',
    wishes: ['Кофемашина', 'Футболка'],
  },

  // === Max_Gamer (id: 3) items ===
  {
    id: 301,
    title: 'Игровой монитор 27"',
    description: '144Hz, 1ms, G-Sync. Идеален для игр.',
    imageUrl: 'https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=800&q=80',
    images: ['https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=800&q=80'],
    category: 'Электроника',
    quantity: 1,
    unit: 'шт',
    authorId: 3,
    holderId: 3,
    isLocked: false,
    createdAt: '2026-08-04T09:00:00Z',
    wishes: ['Игры PS5', 'Клавиатура'],
  },
  {
    id: 302,
    title: 'Геймерское кресло',
    description: 'С подсветкой, регулировкой высоты. Ортопедическое.',
    imageUrl: 'https://images.unsplash.com/photo-1598550476439-c948388916ea?w=800&q=80',
    images: ['https://images.unsplash.com/photo-1598550476439-c948388916ea?w=800&q=80'],
    category: 'Мебель',
    quantity: 1,
    unit: 'шт',
    authorId: 3,
    holderId: 3,
    isLocked: false,
    createdAt: '2026-08-04T10:00:00Z',
    wishes: ['Монитор', 'Мышь'],
  },

  // === Photo_Master (id: 4) items ===
  {
    id: 109,
    title: 'Коллекция книг',
    description: 'Фантастика, 20 томов в отличном состоянии. Азимов, Лем, Брэдбери.',
    imageUrl: 'https://s0.rbk.ru/v6_top_pics/media/img/0/73/347151579745730.webp',
    images: ['https://images.unsplash.com/photo-1512820790803-83ca734da794?w=800&q=80'],
    category: 'Книги',
    quantity: 20,
    unit: 'шт',
    authorId: 4,
    holderId: 4,
    isLocked: false,
    createdAt: '2026-08-05T12:00:00Z',
    wishes: ['Кофемашина', 'Чайный сервиз'],
  },
  {
    id: 401,
    title: 'Фотоаппарат Canon EOS',
    description: 'Полупрофессиональный, с объективом 50mm. Отличное состояние.',
    imageUrl: 'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?w=800&q=80',
    images: ['https://images.unsplash.com/photo-1516035069371-29a1b244cc32?w=800&q=80'],
    category: 'Фотография',
    quantity: 1,
    unit: 'шт',
    authorId: 4,
    holderId: 4,
    isLocked: false,
    createdAt: '2026-08-05T15:00:00Z',
    wishes: ['Штатив', 'Сумка для камеры'],
  },

  // === Music_Lover (id: 5) items ===
  {
    id: 108,
    title: 'Палатка 4-местная',
    description: 'Для кемпинга, водонепроницаемая. Использовали 2 раза.',
    imageUrl: 'https://www.shibargan.ru/wp-content/uploads/2025/08/armejskaja-dvuhslojnaja-vsesezonnaja-palatka-m-10-11.jpg',
    category: 'Туризм',
    quantity: 1,
    unit: 'шт',
    authorId: 5, // Changed from 3 to 5
    holderId: 5,
    isLocked: false,
    createdAt: '2026-08-05T10:00:00Z',
    wishes: ['Гитара', 'Укулеле'],
  },
  {
    id: 110,
    title: 'Скейтборд',
    description: 'Профессиональная доска, колеса мягкие.',
    imageUrl: 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRO7nc6o6b3b68E7tSxFmMqhL4lCP3fcWuCf1KtG8mSMMxPhFYy0noobXaU&s=10',
    category: 'Спорт',
    quantity: 1,
    unit: 'шт',
    authorId: 5,
    holderId: 5,
    isLocked: false,
    createdAt: '2026-08-05T14:00:00Z',
    wishes: ['Защита', 'Кроссовки'],
  },
  {
    id: 501,
    title: 'Виниловый проигрыватель',
    description: 'Ретро стиль, USB оцифровка. Комплект пластинок в подарок.',
    imageUrl: 'https://images.unsplash.com/photo-1563351989-32234e6c3ac8?w=800&q=80',
    images: ['https://images.unsplash.com/photo-1563351989-32234e6c3ac8?w=800&q=80'],
    category: 'Музыка',
    quantity: 1,
    unit: 'шт',
    authorId: 5,
    holderId: 5,
    isLocked: false,
    createdAt: '2026-08-06T11:00:00Z',
    wishes: ['Виниловые пластинки', 'Наушники'],
  },
]);

export const getInitialItems = (): IItem[] => {
  return initialMockItems;
};

// Helper function to get items by userId
export const getItemsByUserId = (userId: number): IItem[] => {
  return initialMockItems.filter(item => item.authorId === userId);
};

// Helper function to get available items for exchange (not locked, currently held by user)
export const getAvailableItemsForUser = (userId: number): IItem[] => {
  return initialMockItems.filter(item => item.holderId === userId && !item.isLocked);
};

export const itemApi = {
  getMyItems: async (userId?: number): Promise<IItem[]> => {
    await new Promise(r => setTimeout(r, 300));
    if (userId) {
      return getItemsByUserId(userId);
    }
    // For backward compatibility, return all items if no userId provided
    return initialMockItems;
  },

  createItem: async (data: Partial<IItem>): Promise<IItem> => {
    await new Promise(r => setTimeout(r, 500));
    // Добавляем новый товар в начало массива (локально)
    const template = initialMockItems[0] || {
      title: data.title || 'Новый предмет',
      description: data.description || '',
      imageUrl: data.imageUrl || '',
      images: data.images || [],
      category: data.category || 'Разное',
      quantity: data.quantity || 1,
      unit: data.unit || 'шт',
      wishes: data.wishes || [],
      authorId: data.authorId || 1,
      holderId: data.holderId || data.authorId || 1,
      isLocked: false,
      createdAt: new Date().toISOString(),
    } as IItem;

    const newItem = {
      ...template,
      ...data,
      id: Date.now(),
      authorId: data.authorId || template.authorId,
      holderId: data.holderId || data.authorId || template.holderId,
      isLocked: false,
      createdAt: new Date().toISOString(),
    } as IItem;

    // Add to internal initialMockItems and persist
    initialMockItems.unshift(newItem);
    saveItemsState(initialMockItems);
    // Notify ItemStore (in-memory) to reload its items
    if (typeof window !== 'undefined' && window.dispatchEvent) {
      window.dispatchEvent(new Event('items_state_changed'));
      console.debug('itemApi: createItem dispatched items_state_changed');
    }
    return newItem;
  },

  searchItems: async (query: string): Promise<IItem[]> => {
    console.log(`Searching: ${query}`);
    await new Promise(r => setTimeout(r, 300));
    return initialMockItems.filter(i =>
      i.title.toLowerCase().includes(query.toLowerCase()) ||
      i.category.toLowerCase().includes(query.toLowerCase())
    );
  },

  transferRight: async (itemId: number, toUserId: number): Promise<void> => {
    console.log(`CHOWN: Item ${itemId} -> User ${toUserId}`);
    await new Promise(r => setTimeout(r, 500));
  },

  addWish: async (itemId: number): Promise<void> => {
    console.log(`Adding wish for item: ${itemId}`);
    await new Promise(r => setTimeout(r, 300));
  },

  updateItem: async (itemId: number, data: Partial<IItem>): Promise<IItem> => {
    await new Promise(r => setTimeout(r, 500));

    const idx = initialMockItems.findIndex(i => i.id === itemId);
    if (idx === -1) {
      throw new Error('Item not found');
    }

    initialMockItems[idx] = { ...initialMockItems[idx], ...data } as IItem;
    saveItemsState(initialMockItems);
    if (typeof window !== 'undefined' && window.dispatchEvent) {
      window.dispatchEvent(new Event('items_state_changed'));
      console.debug('itemApi: updateItem dispatched items_state_changed', itemId, data);
    }

    return initialMockItems[idx];
  },

  lockItem: async (itemId: number): Promise<void> => {
    const idx = initialMockItems.findIndex(i => i.id === itemId);
    if (idx !== -1) {
      initialMockItems[idx].isLocked = true;
      saveItemsState(initialMockItems);
      if (typeof window !== 'undefined' && window.dispatchEvent) {
        window.dispatchEvent(new Event('items_state_changed'));
      }
    }
  },

  unlockItem: async (itemId: number): Promise<void> => {
    const idx = initialMockItems.findIndex(i => i.id === itemId);
    if (idx !== -1) {
      initialMockItems[idx].isLocked = false;
      saveItemsState(initialMockItems);
      if (typeof window !== 'undefined' && window.dispatchEvent) {
        window.dispatchEvent(new Event('items_state_changed'));
      }
    }
  },


  // Создаем новую сделку с цепочкой обмена
  createDeal: async (
    initiatorId: number,
    targetItem: IItem,
    selectedGivingItems: IItem[]
  ): Promise<IExchangeDeal> => {
    await new Promise(r => setTimeout(r, 500));

    // Находим товар, который инициатор хочет получить (targetItem)
    // И формируем цепочку из выбранных товаров которые инициатор отдает
    const chain: IChainLink[] = [];

    // Первое звено - инициатор (отдает свои товары, получает targetItem)
    // Для простоты берем первый товар из selectedGivingItems как основной
    const givingItem = selectedGivingItems[0];

    // Находим кто владеет targetItem (его holderId)
    const targetOwner = mockUsers[targetItem.holderId] || mockUsers[1];

    // Звено инициатора
    chain.push({
      userId: initiatorId,
      user: mockUsers[initiatorId] || mockUsers[1],
      status: ChainLinkStatus.ACCEPTED,
      givingItemId: givingItem.id,
      givingItem: givingItem,
      receivingItemId: targetItem.id,
      receivingItem: targetItem,
      logisticsStatus: LogisticsStatus.NONE,
    });

    // Второе звено - владелец targetItem (отдает targetItem, получает givingItem)
    chain.push({
      userId: targetItem.holderId,
      user: targetOwner,
      status: ChainLinkStatus.PENDING,
      givingItemId: targetItem.id,
      givingItem: targetItem,
      receivingItemId: givingItem.id,
      receivingItem: givingItem,
      logisticsStatus: LogisticsStatus.NONE,
    });

    // Блокируем все товары в сделке и сохраняем состояние
    selectedGivingItems.forEach(item => {
      const idx = initialMockItems.findIndex(i => i.id === item.id);
      if (idx !== -1) {
        initialMockItems[idx].isLocked = true;
      }
    });

    const targetIdx = initialMockItems.findIndex(i => i.id === targetItem.id);
    if (targetIdx !== -1) {
      initialMockItems[targetIdx].isLocked = true;
    }

    saveItemsState(initialMockItems);
    if (typeof window !== 'undefined' && window.dispatchEvent) {
      window.dispatchEvent(new Event('items_state_changed'));
      console.debug('itemApi: createDeal dispatched items_state_changed (locks applied)');
    }

    const newDeal: IExchangeDeal = {
      id: `deal-${Date.now()}`,
      status: DealStatus.CONFIRMING,
      deadline: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      initiatorId,
      chain,
    };

    return newDeal;
  },

  // Передача исключительного права на товар (chown)
  transferExclusiveRight: async (itemId: number, toUserId: number): Promise<void> => {
    console.log(`CHOWN: Item ${itemId} -> User ${toUserId}`);
    await new Promise(r => setTimeout(r, 500));

    const idx = initialMockItems.findIndex(i => i.id === itemId);
    if (idx !== -1) {
      initialMockItems[idx].holderId = toUserId;
      initialMockItems[idx].isLocked = true;
      saveItemsState(initialMockItems);
      if (typeof window !== 'undefined' && window.dispatchEvent) {
        window.dispatchEvent(new Event('items_state_changed'));
      }
    }
  },

};

// src/entities/item/api/itemApi.ts

// TODO: оставить текущий mock как fallback DEV
// export const getInitialItems = () => { ... }

// добавить API helper
/*
const API_BASE = import.meta.env.VITE_API_URL || '';

const apiFetch = async <T>(path: string, options: RequestInit = {}) => {
  const token = localStorage.getItem('access_token');
  const res = await fetch(`${API_BASE}/api/v1${path}`, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
    ...options,
  });
  if (!res.ok) {
    throw new Error(`API ${path} failed: ${await res.text()}`);
  }
  return res.json() as Promise<T>;
};

export const getItems = () => apiFetch<IItem[]>('/items');
export const getItem = (itemId: number) => apiFetch<IItem>(`/items/${itemId}`);
export const getMyItems = (userId: number) => apiFetch<IItem[]>(`/users/${userId}/items`);
export const updateItem = (itemId: number, body: Partial<Pick<IItem,'holderId'|'isLocked'>>) =>
  apiFetch<IItem>(`/items/${itemId}`, { method: 'PATCH', body: JSON.stringify(body) });
export const createItem = (item: Partial<IItem>) =>
  apiFetch<IItem>('/items', { method: 'POST', body: JSON.stringify(item) });

export const dealApi = {
  getDeals: () => apiFetch<IExchangeDeal[]>('/deals'),
  getDeal: (dealId: string) => apiFetch<IExchangeDeal>(`/deals/${dealId}`),
  createDeal: (payload: {
    initiatorId: number;
    targetItemId: number;
    selectedGivingItemIds: number[];
    dealType: 'DIRECT' | 'CHAIN';
  }) => apiFetch<IExchangeDeal>('/deals', { method: 'POST', body: JSON.stringify(payload) }),
  confirmDeal: (dealId: string) => apiFetch<void>(`/deals/${dealId}/confirm`, { method: 'POST' }),
  cancelDeal: (dealId: string, reason?: string) => apiFetch<void>(`/deals/${dealId}/cancel`, { method: 'POST', body: JSON.stringify({ reason }) }),
  transferExclusiveRight: (itemId: number, toUserId: number, lockItem: boolean = false) =>
    apiFetch<IItem>(`/items/${itemId}/rights`, { method: 'PATCH', body: JSON.stringify({ holderId: toUserId, isLocked: lockItem }) }),
};
*/