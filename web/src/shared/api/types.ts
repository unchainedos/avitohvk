
export const UserRole = {
  USER: 'USER',
  ADMIN: 'ADMIN',
} as const;
export type UserRole = (typeof UserRole)[keyof typeof UserRole];

// Статусы звеньев (участников) в цепочке
export const ChainLinkStatus = {
  WAITING: 'WAITING',     // Ждет своей очереди
  PENDING: 'PENDING',     // На рассмотрении
  ACCEPTED: 'ACCEPTED',   // Подтвердил
  DECLINED: 'DECLINED',   // Отказался
} as const;
export type ChainLinkStatus = (typeof ChainLinkStatus)[keyof typeof ChainLinkStatus];

// Статусы всей сделки
export const DealStatus = {
  PENDING: 'PENDING',    
  CONFIRMING: 'CONFIRMING',               
  WAITING_FOR_REQUIRED_USER: 'WAITING_FOR_REQUIRED_USER', 
  CONFIRMED: 'CONFIRMED',               // Все подтвердили -> логистика
  ACTIVE: 'ACTIVE',                     // Добавил для удобства UI (синоним CONFIRMED в процессе логистики)
  CANCELLED: 'CANCELLED',               
  COMPLETED: 'COMPLETED',               
} as const;
export type DealStatus = (typeof DealStatus)[keyof typeof DealStatus];

// Статусы логистики (ПВЗ)
export const LogisticsStatus = {
  NONE: 'NONE',
  PENDING_DROP_OFF: 'PENDING_DROP_OFF', 
  DROPPED_OFF: 'DROPPED_OFF',           
  IN_TRANSIT: 'IN_TRANSIT',             
  DELIVERED_TO_PVZ: 'DELIVERED_TO_PVZ', 
  COMPLETED: 'COMPLETED',               
} as const;
export type LogisticsStatus = (typeof LogisticsStatus)[keyof typeof LogisticsStatus];

export interface IUser {
  id: number;
  username: string;
  rating: number;        
  declineCount: number;  
  avatarUrl?: string;
  pvzAddress?: string;   
}

export interface IItem {
  id: number;
  title: string;
  description: string;
  imageUrl: string;
  images?: string[];
  category: string;
  quantity: number;
  unit: string;
  wishes?: string[];
  
  authorId: number;      
  holderId: number;      // Исключительное право
  
  isLocked: boolean;     
  createdAt: string;
}

export interface IChainLink {
  userId: number;
  user: IUser;
  status: ChainLinkStatus;
  
  givingItemId: number; 
  givingItem: IItem;
  
  receivingItemId?: number;
  receivingItem?: IItem;

  // Добавляем статус логистики прямо в звено для удобства рендера
  logisticsStatus: LogisticsStatus; 
}

export interface IExchangeDeal {
  id: string;
  status: DealStatus;
  deadline: string; 
  
  // Порядок: 0 - инициатор (хочет получить), N - владелец цели (отдает)
  chain: IChainLink[]; 
  
  initiatorId: number; // ID пользователя, который запустил подбор (Саша)
  declineReason?: string; // Причина отмены
}

// Для общего трекинга маршрута (опционально)
export interface ILogisticsStep {
  itemId: number;
  status: LogisticsStatus;
  fromPvz: string;
  toPvz: string;
  updatedAt: string;
}

export interface ApiResponse<T> {
  data: T;
  message?: string;
}