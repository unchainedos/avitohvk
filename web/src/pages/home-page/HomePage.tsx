// src/pages/home-page/HomePage.tsx
import { useState } from 'react';
import { observer } from 'mobx-react-lite';
import { ItemCard } from '../../entities/item/ui/ItemCard';
import { itemStore } from '../../app/providers/ItemStore';
import { dealStore } from '../../app/providers/DealStore';
import { useAuthStore } from '../../app/hooks/useAuthStore';

export const HomePage = observer(() => {
  const authStore = useAuthStore();
  const currentUserId = authStore.user?.id;

  const [searchQuery, setSearchQuery] = useState('');

  const allItems = itemStore.all.filter(i => {
    if (dealStore.isItemReserved(i.id)) {
      return false;
    }

    if (currentUserId !== undefined && currentUserId !== null) {
      if (i.isLocked && i.holderId !== currentUserId) {
        return false;
      }
      if (i.authorId === currentUserId || i.holderId === currentUserId) {
        return false;
      }
      if (i.holderId !== i.authorId) {
        return false;
      }
      return true;
    }
    if (i.holderId !== i.authorId) {
      return false;
    }
    return !i.isLocked;
  });

  const items = searchQuery.trim()
    ? allItems.filter(i => i.title.toLowerCase().includes(searchQuery.trim().toLowerCase()) || i.category.toLowerCase().includes(searchQuery.trim().toLowerCase()))
    : allItems;

  const handleSearch = () => {
    setSearchQuery(searchQuery.trim());
  };

  return (
    <div className="space-y-8 w-full pb-20">

      {/* Hero Search */}
      <div className="bg-white p-8 md:p-12 rounded-2xl shadow-sm border border-gray-100 w-full flex flex-col items-center">
        <h1 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4 text-center">Найти обмен</h1>

        {/* Принудительное центрирование текста */}
        <p className="text-gray-500 mb-8 max-w-2xl text-center mx-auto">
          Введите название товара, который вы хотите получить, или смотрите все доступные варианты ниже.
        </p>

        <div className="w-full max-w-3xl flex flex-col sm:flex-row gap-3 justify-center">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            placeholder="Например: Велосипед, Апельсин..."
            className="flex-1 px-6 py-4 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00AAFF] focus:border-transparent text-lg shadow-inner text-center sm:text-left"
          />
          <button
            onClick={handleSearch}
            className="px-10 py-4 bg-gray-900 text-white rounded-xl font-bold hover:bg-gray-800 transition-colors text-lg shadow-lg hover:shadow-xl transform hover:-translate-y-0.5"
          >
            Найти
          </button>
        </div>
      </div>

      {/* Результаты */}
      <div className="w-full">
          <div className="flex items-center justify-between mb-6 px-2">
              <h2 className="text-xl font-bold text-gray-900">
                  {searchQuery ? `Результаты: "${searchQuery}"` : 'Все доступные обмены'}
              </h2>
              <span className="text-sm text-gray-500 bg-gray-100 px-3 py-1 rounded-full">
                  Найдено: {items.length}
              </span>
          </div>

          {items.length > 0 ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6 w-full">
                  {items.map(item => (
                      <ItemCard key={item.id} item={item} />
                  ))}
              </div>
          ) : (
              // Блок "Ничего не найдено"
              <div className="text-center py-20 text-gray-400 bg-white rounded-2xl border border-dashed border-gray-200 w-full flex flex-col items-center justify-center">
                  <div className="text-6xl mb-4 opacity-50">🔍</div>
                  <p className="text-xl font-bold text-gray-600 mb-2">Ничего не найдено</p>
                  <p className="text-sm text-gray-400 max-w-md mx-auto">
                      По вашему запросу "{searchQuery}" нет товаров. Попробуйте изменить название или добавьте свой товар в профиле, чтобы запустить цепочку.
                  </p>
                  <button
                    onClick={() => setSearchQuery('')}
                    className="mt-6 text-[#00AAFF] font-medium hover:underline"
                  >
                      Сбросить поиск
                  </button>
              </div>
          )}
      </div>
    </div>
  );
});