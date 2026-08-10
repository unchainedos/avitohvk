// src/entities/user/api/userApi.ts
import type { IUser } from '../../../shared/api/types';
import { mockUsersDb } from '../../../shared/api/authApi';

// Build mockUsers record from authApi mock database
export const mockUsers: Record<number, IUser> = {};
for (const record of Object.values(mockUsersDb)) {
  mockUsers[record.user.id] = record.user;
}

// Fallback for legacy IDs if needed
const legacyUsers: Record<number, IUser> = {
  1: { id: 1, username: 'Alex_Dev', rating: 4.8, declineCount: 1 },
  2: { id: 2, username: 'Dima_Biker', rating: 4.9, declineCount: 0 },
  3: { id: 3, username: 'Max_Gamer', rating: 4.5, declineCount: 2 },
  4: { id: 4, username: 'Photo_Master', rating: 5.0, declineCount: 0 },
  5: { id: 5, username: 'Music_Lover', rating: 4.7, declineCount: 0 },
};

// Merge legacy users (only if not already in mockUsersDb)
for (const [id, user] of Object.entries(legacyUsers)) {
  if (!mockUsers[Number(id)]) {
    mockUsers[Number(id)] = user;
  }
}

export const userApi = {
  getUserById: async (id: number): Promise<IUser> => {
    await new Promise(r => setTimeout(r, 100));
    return mockUsers[id] || mockUsers[1];
  }
};