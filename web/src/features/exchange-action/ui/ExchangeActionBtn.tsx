// src/features/exchange-action/ui/ExchangeActionBtn.tsx
import React, { useState } from 'react';
import type { IItem } from '../../../shared/api/types';
import { useAuthStore } from '../../../app/hooks/useAuthStore';
import { itemStore } from '../../../app/providers/ItemStore';
import { DirectExchangeModal } from './DirectExchangeModal';
import { JoinChainModal } from '../../join-chain/ui/JoinChainModal';

interface ExchangeActionBtnProps {
  item: IItem;
  onDealCreated?: () => void;
}

export const ExchangeActionBtn: React.FC<ExchangeActionBtnProps> = ({ item, onDealCreated }) => {
  const authStore = useAuthStore();
  const [isDirectModalOpen, setIsDirectModalOpen] = useState(false);
  const [isChainModalOpen, setIsChainModalOpen] = useState(false);

  const currentUser = authStore.user;
  const isAuthenticated = authStore.isAuthenticated;

  // Get user's available items for exchange
  const myAvailableItems = currentUser ? itemStore.all.filter(item => item.holderId === currentUser.id && !item.isLocked) : [];
  const hasItemsToExchange = myAvailableItems.length > 0;

  // Check if I have what the owner wants (for direct exchange)
  const hasDesiredItem = item.wishes?.some(wish =>
    myAvailableItems.some(myItem =>
      myItem.title.toLowerCase().includes(wish.toLowerCase()) ||
      wish.toLowerCase().includes(myItem.title.toLowerCase())
    )
  );

  // Determine which button to show
  const isMyItem = currentUser && (item.holderId === currentUser.id || item.authorId === currentUser.id);

  if (isMyItem || item.isLocked) {
    return null;
  }

  const handleClick = () => {
    if (!isAuthenticated) {
      // Redirect to auth will be handled by parent or navigate
      return;
    }

    if (hasDesiredItem) {
      // Scenario A: Direct exchange possible
      setIsDirectModalOpen(true);
    } else {
      // Scenario B: No direct match -> Chain
      if (!hasItemsToExchange) {
        // Show message about needing items
        alert('У вас нет товаров для обмена. Добавьте товар в профиле чтобы участвовать в цепочке.');
        return;
      }
      setIsChainModalOpen(true);
    }
  };

  const disabled = !isAuthenticated;

  return (
    <>
      {/* Single smart button */}
      <button
        onClick={handleClick}
        disabled={disabled}
        className={`w-full px-6 py-3 rounded-xl font-bold text-white shadow-lg transition-all flex flex-col items-center justify-center ${
          hasDesiredItem && isAuthenticated
            ? 'bg-green-500 hover:bg-green-600 shadow-green-200 hover:-translate-y-0.5'
            : isAuthenticated && hasItemsToExchange
            ? 'bg-[#00AAFF] hover:bg-[#0095E0] shadow-blue-200 hover:-translate-y-0.5'
            : 'bg-gray-300 cursor-not-allowed shadow-none'
        }`}
      >
        <span className="text-xs opacity-90 font-medium">
          {isAuthenticated
            ? (hasDesiredItem
                ? `${myAvailableItems.length} товар(а) подходит`
                : hasItemsToExchange
                  ? `${myAvailableItems.length} товар(а) доступно`
                  : 'Нет товаров')
            : 'Требуется вход'}
        </span>
        <span className="text-sm">
          {hasDesiredItem ? 'Предложить обмен' : 'Встать в цепочку'}
        </span>
      </button>

      {!isAuthenticated && (
        <p className="text-xs text-center text-gray-500 mt-1">
          Войдите чтобы участвовать в обмене
        </p>
      )}

      {/* Direct Exchange Modal */}
      {isDirectModalOpen && (
        <DirectExchangeModal
          targetItem={item}
          onClose={() => setIsDirectModalOpen(false)}
          onConfirm={() => {
            setIsDirectModalOpen(false);
            onDealCreated?.();
          }}
        />
      )}

      {/* Join Chain Modal */}
      {isChainModalOpen && (
        <JoinChainModal
          targetItem={item}
          onClose={() => setIsChainModalOpen(false)}
          onConfirm={() => {
            setIsChainModalOpen(false);
            onDealCreated?.();
          }}
        />
      )}
    </>
  );
};