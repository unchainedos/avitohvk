// src/features/exchange-action/ui/DirectExchangeModal.tsx
import { useState } from 'react';
import type { IItem, IExchangeDeal } from '../../../shared/api/types';
import { useAuthStore } from '../../../app/hooks/useAuthStore';
import { dealStore } from '../../../app/providers/DealStore';
import { itemStore } from '../../../app/providers/ItemStore';

interface DirectExchangeModalProps {
  targetItem: IItem; // The item we want to receive
  onClose: () => void;
  onConfirm: (deal: IExchangeDeal) => void;
}

export const DirectExchangeModal = ({ targetItem, onClose, onConfirm }: DirectExchangeModalProps) => {
  const authStore = useAuthStore();
  const currentUserId = authStore.user?.id ?? 1;

  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const myItems = itemStore.all.filter(i =>
    i.holderId === currentUserId &&
    i.id !== targetItem.id &&
    !i.isLocked &&
    !dealStore.isItemReserved(i.id)
  );

  const toggleSelection = (id: number) => {
    setSelectedIds(prev =>
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    );
  };

  const handleConfirm = async () => {
    if (selectedIds.length === 0) return;

    setIsSubmitting(true);
    try {
      // Get full data for selected items
      const selectedItems = myItems.filter(item => selectedIds.includes(item.id));

      // Create deal through DealStore
      const newDeal = await dealStore.createDeal(
        currentUserId,
        targetItem,
        selectedItems,
        'DIRECT'
      );

      // Close modal
      onClose();
      // Call onConfirm with created deal
      onConfirm(newDeal);
    } catch (error) {
      console.error(error);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 backdrop-blur-sm">
        <div className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden flex flex-col">

          {/* Header */}
          <div className="p-6 border-b border-gray-100 bg-green-50">
            <h2 className="text-xl font-bold text-gray-900">Предложить прямой обмен</h2>
            <p className="text-sm text-gray-600 mt-1">
              Вы хотите получить: <span className="font-semibold text-green-700">{targetItem.title}</span>
            </p>
            <p className="text-xs text-gray-500 mt-2">
              Владелец этого товара хочет: <span className="font-medium">{targetItem.wishes?.join(', ')}</span>
            </p>
          </div>

          {/* Body */}
          <div className="p-6 overflow-y-auto flex-1">
            <p className="text-sm text-gray-600 mb-4">
              Выберите один из ваших товаров, который подходит владельцу:
            </p>

            {myItems.length === 0 ? (
              <div className="text-center py-8 text-gray-400">
                У вас нет подходящих товаров для прямого обмена.
              </div>
            ) : (
              <div className="space-y-3">
                {myItems.map(item => {
                  // Check if this item matches owner's wishes
                  const isMatch = targetItem.wishes?.some(wish =>
                    item.title.toLowerCase().includes(wish.toLowerCase()) ||
                    wish.toLowerCase().includes(item.title.toLowerCase())
                  );

                  return (
                    <div
                      key={item.id}
                      onClick={() => isMatch && toggleSelection(item.id)}
                      className={`
                        flex items-center gap-4 p-3 rounded-xl border cursor-pointer transition-all
                        ${!isMatch ? 'opacity-50 cursor-not-allowed' : ''}
                        ${selectedIds.includes(item.id)
                          ? 'border-green-500 bg-green-50 ring-1 ring-green-500'
                          : isMatch
                            ? 'border-gray-200 hover:bg-gray-50'
                            : 'border-gray-100 bg-gray-50'}
                      `}
                    >
                      {/* Checkbox visual */}
                      <div className={`
                        w-5 h-5 rounded border flex items-center justify-center shrink-0
                        ${!isMatch ? 'border-gray-200 bg-gray-100' : selectedIds.includes(item.id)
                          ? 'bg-green-500 border-green-500'
                          : 'border-gray-300'}
                      `}>
                        {selectedIds.includes(item.id) && (
                          <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                          </svg>
                        )}
                      </div>

                      {/* Mini Item Info */}
                      <img src={item.imageUrl} alt="" className="w-12 h-12 rounded-lg object-cover bg-gray-100" />
                      <div className="flex-1 min-w-0">
                        <h4 className="font-medium text-gray-900 truncate">{item.title}</h4>
                        <p className="text-xs text-gray-500 truncate">{item.category}</p>
                        {isMatch && (
                          <span className="text-xs text-green-600 font-medium">✓ Подходит владельцу</span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="p-6 border-t border-gray-100 bg-gray-50 flex justify-end gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-gray-600 font-medium hover:text-gray-900 transition-colors"
            >
              Отмена
            </button>
            <button
              onClick={handleConfirm}
              disabled={selectedIds.length === 0 || isSubmitting}
              className="px-6 py-2 bg-green-500 text-white rounded-lg font-medium hover:bg-green-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
            >
              {isSubmitting ? 'Отправка...' : `Отправить заявку (${selectedIds.length})`}
            </button>
          </div>
        </div>
      </div>
    </>
  );
};