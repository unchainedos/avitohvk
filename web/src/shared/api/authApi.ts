// src/shared/api/authApi.ts
import type { IUser, ApiResponse } from './types';
// Backend integration notes:
// import { apiClient } from './client';

interface LoginRequest {
  username: string;
  password: string;
}

interface RegisterRequest {
  username: string;
  password: string;
  email: string;
}

interface ChangeProfileRequest {
  oldPassword?: string;
  newPassword?: string;
  newUsername?: string;
  newAvatarUrl?: string;
}

interface AuthResponse {
  token: string;
  user: IUser;
}

// Mock users database with unique credentials, items and wishes
export const mockUsersDb: Record<string, { password: string; email: string; user: IUser; wishes?: string[] }> = {
  alex: {
    password: '123456',
    email: 'alex@example.com',
    user: { id: 1, username: 'Alex_Dev', rating: 4.8, declineCount: 1, pvzAddress: 'ПВЗ №123, ул. Ленина 10' },
    wishes: ['Смартфон', 'Наушники', 'Книги', 'Винил'],
  },
  dima: {
    password: '123456',
    email: 'dima@example.com',
    user: { id: 2, username: 'Dima_Biker', rating: 4.9, declineCount: 0, pvzAddress: 'ПВЗ №45, пр. Мира 5' },
    wishes: ['Апельсины', 'Лодка', 'Гитара', 'Кофемашина', 'Футболка'],
  },
  max: {
    password: '123456',
    email: 'max@example.com',
    user: { id: 3, username: 'Max_Gamer', rating: 4.5, declineCount: 2, pvzAddress: 'ПВЗ №78, ул. Гагарина 15' },
    wishes: ['Игры PS5', 'Клавиатура', 'Монитор', 'Мышь'],
  },
  photo: {
    password: '123456',
    email: 'photo@example.com',
    user: { id: 4, username: 'Photo_Master', rating: 5.0, declineCount: 0, pvzAddress: 'ПВЗ №12, ул. Пушкина 8' },
    wishes: ['Кофемашина', 'Чайный сервиз', 'Штатив', 'Сумка для камеры'],
  },
  music: {
    password: '123456',
    email: 'music@example.com',
    user: { id: 5, username: 'Music_Lover', rating: 4.7, declineCount: 0, pvzAddress: 'ПВЗ №34, ул. Лермонтова 22' },
    wishes: ['Гитара', 'Укулеле', 'Защита', 'Кроссовки', 'Виниловые пластинки'],
  },
};

// Mock JWT token generator
const generateMockToken = (username: string): string => {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(JSON.stringify({
    username,
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + 86400 // 24 hours
  }));
  const signature = btoa('mock-signature');
  return `${header}.${payload}.${signature}`;
};

// Mock delay simulation
const mockDelay = (ms: number): Promise<void> => new Promise(resolve => setTimeout(resolve, ms));

export const authApi = {
  /**
   * POST /api/v1/login
   * Mock implementation - replace with axios call when backend is ready
   */
  login: async ({ username, password }: LoginRequest): Promise<ApiResponse<AuthResponse>> => {
    await mockDelay(800);

    // Simple validation mock
    if (!username || !password) {
      throw new Error('Username and password are required');
    }

    // Check against mock database (case-insensitive)
    const lowerUsername = username.toLowerCase();
    const userRecord = mockUsersDb[lowerUsername];

    if (!userRecord) {
      throw new Error('User not found. Please register first.');
    }

    if (userRecord.password !== password) {
      throw new Error('Invalid password');
    }

    // Generate mock response
    const token = generateMockToken(username);
    const user: IUser = { ...userRecord.user };

    return {
      data: { token, user },
      message: 'Login successful',
    };
  },

  /**
   * POST /api/v1/register
   * Mock implementation - replace with axios call when backend is ready
   */
  register: async ({ username, password, email }: RegisterRequest): Promise<ApiResponse<AuthResponse>> => {
    await mockDelay(800);

    // Simple validation mock
    if (!username || !password || !email) {
      throw new Error('Username, password and email are required');
    }

    // Email format validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      throw new Error('Invalid email format');
    }

    // Check if username already exists
    const lowerUsername = username.toLowerCase();
    if (mockUsersDb[lowerUsername]) {
      throw new Error('Username already exists');
    }

    // Create new user
    const newUser: IUser = {
      id: Date.now(),
      username,
      rating: 5.0,
      declineCount: 0,
      avatarUrl: undefined,
      pvzAddress: undefined,
    };

    // Add to mock database (in-memory only, will reset on page reload)
    mockUsersDb[lowerUsername] = {
      password,
      email,
      user: newUser,
    };

    // Generate mock response
    const token = generateMockToken(username);

    return {
      data: { token, user: newUser },
      message: 'Registration successful',
    };
  },

  /**
   * Get user by ID (for profile pages)
   */
  getUserById: async (id: number): Promise<IUser | null> => {
    await mockDelay(100);
    for (const record of Object.values(mockUsersDb)) {
      if (record.user.id === id) {
        return record.user;
      }
    }
    return null;
  },

  /**
   * PUT /api/v1/profile
   * Change username and/or password
   */
  changeProfile: async (userId: number, { oldPassword, newPassword, newUsername, newAvatarUrl }: ChangeProfileRequest): Promise<ApiResponse<IUser>> => {
    await mockDelay(500);

    // Find user record by ID
    let userRecord: typeof mockUsersDb[string] | undefined;
    let usernameKey: string | undefined;

    for (const [key, record] of Object.entries(mockUsersDb)) {
      if (record.user.id === userId) {
        userRecord = record;
        usernameKey = key;
        break;
      }
    }

    if (!userRecord) {
      throw new Error('User not found');
    }

    // If changing password, verify old password
    if (newPassword) {
      if (!oldPassword) {
        throw new Error('Old password is required to change password');
      }
      if (oldPassword !== userRecord.password) {
        throw new Error('Invalid old password');
      }
    }

    // If changing username, check if it's already taken
    if (newUsername && newUsername.toLowerCase() !== usernameKey) {
      if (mockUsersDb[newUsername.toLowerCase()]) {
        throw new Error('Username already exists');
      }
    }

    // Apply changes
    if (newPassword) {
      userRecord.password = newPassword;
    }

    if (newUsername) {
      // Update username in the user object
      userRecord.user.username = newUsername;
      // Move to new key if username changed
      if (usernameKey && newUsername.toLowerCase() !== usernameKey) {
        mockUsersDb[newUsername.toLowerCase()] = userRecord;
        delete mockUsersDb[usernameKey];
      }
    }

    if (newAvatarUrl !== undefined) {
      userRecord.user.avatarUrl = newAvatarUrl || undefined;
    }

    return {
      data: userRecord.user,
      message: 'Profile updated successfully',
    };
  },

  /**
   * Get user wishes by user ID
   */
  getUserWishes: async (userId: number): Promise<string[]> => {
    await mockDelay(100);
    for (const record of Object.values(mockUsersDb)) {
      if (record.user.id === userId) {
        return record.wishes || [];
      }
    }
    return [];
  },

  /**
   * Add wish to user
   */
  addWish: async (userId: number, wish: string): Promise<string[]> => {
    await mockDelay(200);
    for (const record of Object.values(mockUsersDb)) {
      if (record.user.id === userId) {
        if (!record.wishes) {
          record.wishes = [];
        }
        if (!record.wishes.includes(wish)) {
          record.wishes.push(wish);
        }
        return record.wishes;
      }
    }
    return [];
  },

  /**
   * Remove wish from user
   */
  removeWish: async (userId: number, wish: string): Promise<string[]> => {
    await mockDelay(200);
    for (const record of Object.values(mockUsersDb)) {
      if (record.user.id === userId) {
        if (record.wishes) {
          record.wishes = record.wishes.filter(w => w !== wish);
        }
        return record.wishes || [];
      }
    }
    return [];
  },
};

// Ready-to-use axios implementation (uncomment and comment out mock above when backend is ready):
/*
export const authApi = {
  login: async ({ username, password }: LoginRequest): Promise<ApiResponse<AuthResponse>> => {
    const response = await apiClient.post<ApiResponse<AuthResponse>>('/v1/login', { username, password });
    return response.data;
  },

  register: async ({ username, password, email }: RegisterRequest): Promise<ApiResponse<AuthResponse>> => {
    const response = await apiClient.post<ApiResponse<AuthResponse>>('/v1/register', { username, password, email });
    return response.data;
  },
};
*/