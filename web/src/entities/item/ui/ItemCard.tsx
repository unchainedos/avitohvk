// src/entities/item/ui/ItemCard.tsx
import { useNavigate } from 'react-router-dom';
import type { IItem } from '../../../shared/api/types';
import { UserBadge } from '../../user/ui/UserBadge';
import { mockUsers } from '../../user/api/userApi';

interface ItemCardProps {
  item: IItem;
  currentUserId?: number;
}

export const ItemCard = ({ item, currentUserId }: ItemCardProps) => {
  const navigate = useNavigate();
  const isRightTransferred = item.holderId !== item.authorId;
  const isHolder = currentUserId !== undefined && item.holderId === currentUserId;
  const showExclusiveBadge = isRightTransferred && isHolder;
  const labelTitle = isHolder ? 'Распорядитель' : 'Владелец';
  const holder = mockUsers[item.holderId] || { id: 0, username: 'Unknown', rating: 0, declineCount: 0 };

  const handleClick = () => {
    navigate(`/item/${item.id}`);
  };

  return (
    <div
      onClick={handleClick}
      className="cursor-pointer relative bg-white rounded-xl border border-gray-200 overflow-hidden shadow-sm hover:shadow-md transition-all group flex flex-col h-full"
    >


      {showExclusiveBadge && (
        <div className="absolute top-2 left-2 bg-purple-600 text-white text-xs font-bold px-2 py-1 rounded z-10 shadow-sm">
          ⚡ Исключительное право
        </div>
      )}

      <div className="h-48 bg-gray-100 w-full relative overflow-hidden">
        <img
          src={item.imageUrl}
          alt={item.title}
          className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
        />

      </div>

      <div className="p-4 flex flex-col flex-grow">
        <div className="mb-3">
          <h3 className="font-bold text-lg text-gray-900 line-clamp-1 group-hover:text-[#00AAFF] transition-colors">{item.title}</h3>
          <p className="text-sm text-gray-500 line-clamp-2 mt-1 h-10">{item.description}</p>
        </div>

        {/* Блок желаний владельца (вместо кнопки) */}
        {item.wishes && item.wishes.length > 0 && (
            <div className="mb-3 p-2 bg-blue-50 rounded-lg border border-blue-100">
                <span className="text-[10px] uppercase font-bold text-blue-600 block mb-1">Владелец хочет:</span>
                <div className="flex flex-wrap gap-1">
                    {item.wishes.slice(0, 2).map((w, i) => (
                        <span key={i} className="text-xs font-medium text-blue-800 bg-white px-1.5 py-0.5 rounded border border-blue-50">
                            {w}
                        </span>
                    ))}
                    {item.wishes.length > 2 && <span className="text-xs text-blue-400">+{item.wishes.length - 2}</span>}
                </div>
            </div>
        )}

        <div className="flex items-center justify-between text-xs text-gray-500 mb-4 mt-auto">
          <span className="bg-gray-100 px-2 py-0.5 rounded">{item.category}</span>
          <span>{item.quantity} {item.unit}</span>
        </div>

        <div className="pt-3 border-t border-gray-100">
          <div className="text-[10px] uppercase text-gray-400 font-semibold mb-1">
            {labelTitle}
          </div>
          <UserBadge user={holder} size="sm" />
        </div>

        {/* Убрали кнопки действий отсюда */}
      </div>
    </div>
  );
};