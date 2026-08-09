// src/entities/user/ui/UserBadge.tsx
import type { IUser } from '../../../shared/api/types';

interface UserBadgeProps {
  user: IUser;
  size?: 'sm' | 'md';
}

export const UserBadge = ({ user, size = 'md' }: UserBadgeProps) => {
  const sizeClasses = size === 'sm' ? 'text-xs gap-1' : 'text-sm gap-2';
  const avatarSize = size === 'sm' ? 'w-6 h-6' : 'w-8 h-8';

  return (
    <div className={`flex items-center ${sizeClasses}`}>
      <div className={`${avatarSize} rounded-full bg-gray-100 flex items-center justify-center font-bold text-gray-600 border border-gray-200 overflow-hidden`}>
        {user.avatarUrl ? (
          <img src={user.avatarUrl} alt="" className="w-full h-full object-cover" />
        ) : (
          user.username.charAt(0)
        )}
      </div>
      <div className="flex flex-col leading-none">
        <span className="font-medium text-gray-900">{user.username}</span>
        <div className="flex items-center gap-1 mt-0.5">
          <span className="text-yellow-500 font-bold">★ {user.rating}</span>
          {user.declineCount > 0 && (
            <span className="text-red-400 text-[10px]" title="Отказов от сделок">
              ⚠ {user.declineCount}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};