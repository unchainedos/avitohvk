// src/widgets/chain-visualizer/ui/ChainVisualizer.tsx
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

// Импорты типов и компонентов
import { DealStatus } from '../../../shared/api/types';
import type { IExchangeDeal, IChainLink, IItem } from '../../../shared/api/types';
import { DeclineModal } from '../../../features/decline-deal/ui/DeclineModal';

interface ChainVisualizerProps {
  deal: IExchangeDeal;
  currentUserId: number;
}

// Вспомогательный компонент для карточки товара (кликабельной)
const ItemCard: React.FC<{ item: IItem; label: string; color: 'slate' | 'blue' }> = ({ item, label, color }) => {
  const navigate = useNavigate();

  // Останавливаем всплытие, чтобы клик по товару не открывал профиль юзера
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigate(`/item/${item.id}`); // Переход на страницу товара
  };

  const borderClass = color === 'blue' ? 'border-blue-100 bg-blue-50' : 'border-slate-200 bg-white';
  const textClass = color === 'blue' ? 'text-blue-900' : 'text-slate-700';
  const labelClass = color === 'blue' ? 'text-blue-400' : 'text-slate-400';
  const barClass = color === 'blue' ? 'bg-blue-400' : 'bg-slate-300';

  return (
    <div
      onClick={handleClick}
      className={`relative overflow-hidden rounded-xl p-3 border shadow-sm cursor-pointer hover:shadow-md transition-all group/item ${borderClass}`}
    >
      <div className={`absolute top-0 left-0 w-1 h-full ${barClass}`}></div>
      <p className={`text-[10px] uppercase font-bold mb-2 pl-2 ${labelClass}`}>{label}</p>
      <div className="flex items-center gap-3 pl-2">
        <div className="w-10 h-10 rounded-lg bg-white flex-shrink-0 overflow-hidden shadow-sm border border-slate-100">
          <img src={item.imageUrl} alt={item.title} className="w-full h-full object-cover" />
        </div>
        <div className="min-w-0">
          <span className={`text-sm font-medium truncate block ${textClass} group-hover/item:underline`}>
            {item.title}
          </span>
          <span className="text-xs text-slate-400">x{item.quantity} {item.unit}</span>
        </div>
      </div>
    </div>
  );
};

export const ChainVisualizer: React.FC<ChainVisualizerProps> = ({ deal, currentUserId }) => {
  const navigate = useNavigate();
  const [isDeclineModalOpen, setIsDeclineModalOpen] = useState(false);

  const currentUserLink = deal.chain.find((link) => link.userId === currentUserId);

  const handleFindNewDeal = () => {
    setIsDeclineModalOpen(false);
    navigate('/');
  };

  // --- Сценарий: Сделка отменена ---
  if (deal.status === DealStatus.CANCELLED) {
    return (
      <>
        <div className="flex flex-col items-center justify-center p-12 bg-red-50 rounded-3xl border-2 border-dashed border-red-200 text-center">
          <div className="text-6xl mb-4 filter grayscale opacity-80">🥀</div>
          <h2 className="text-3xl font-bold text-red-600 mb-2">Обмен отменен</h2>
          <p className="text-slate-600 mb-8 max-w-md">
            Цепочка была разорвана. Все исключительные права возвращены владельцам.
          </p>
          <button
            onClick={() => setIsDeclineModalOpen(true)}
            className="px-6 py-3 bg-white border border-red-200 text-red-600 rounded-xl font-medium hover:bg-red-50 transition shadow-sm"
          >
            Почему это произошло?
          </button>
        </div>

        <DeclineModal
          isOpen={isDeclineModalOpen}
          onClose={() => setIsDeclineModalOpen(false)}
          reason={deal.declineReason || 'Участник отказался от обмена'}
          initiatorName={deal.chain.find(l => l.status === 'DECLINED')?.user.username || 'Неизвестный'}
          onFindNewDeal={handleFindNewDeal}
        />
      </>
    );
  }

  return (
    <div className="w-full space-y-8">
      {/* --- Визуализация цепи --- */}
      <div className="overflow-x-auto pb-8 pt-2 scrollbar-hide">
        <div className="flex items-start min-w-max md:min-w-0 md:flex-wrap md:justify-center gap-4 md:gap-0">

          {deal.chain.map((link: IChainLink, index: number) => {
            const isLast = index === deal.chain.length - 1;
            const isInitiator = link.userId === deal.initiatorId;

            // Определяем, что получает этот участник.
            // В линейной цепи: Участник N получает то, что отдает Участник N-1.
            // Для первого участника (инициатора) он получает то, что отдает последний (замыкание).
            let receivingItem: IItem | undefined;
            let receivingLabel = "Получает";

            if (index === 0) {
              // Первый получает от последнего
              receivingItem = deal.chain[deal.chain.length - 1].givingItem;
            } else {
              // Остальные получают от предыдущего
              receivingItem = deal.chain[index - 1].givingItem;
            }

            return (
              <React.Fragment key={`${link.userId}-${index}`}>
                {/* Колонка участника */}
                <div className="flex flex-col items-center w-64">

                  {/* Шапка профиля (Кликабельная область -> Профиль) */}
                  <div
                    className="flex flex-col items-center mb-4 cursor-pointer group/user"
                    onClick={() => navigate(`/user/${link.userId}`)}
                  >
                    <div className="relative mb-2">
                      <div className="w-16 h-16 rounded-full bg-gradient-to-br from-slate-200 to-slate-300 flex items-center justify-center text-slate-600 text-xl font-bold shadow-md border-4 border-white overflow-hidden group-hover/user:ring-2 ring-blue-400 transition-all">
                        {link.user.avatarUrl ? (
                          <img src={link.user.avatarUrl} alt="" className="w-full h-full object-cover" />
                        ) : (
                          link.user.username.charAt(0).toUpperCase()
                        )}
                      </div>
                      <div className="absolute -bottom-1 -right-1 bg-yellow-400 text-xs px-2 py-0.5 rounded-full font-bold border-2 border-white text-yellow-900 shadow-sm flex items-center gap-0.5">
                        ★ {link.user.rating}
                      </div>
                    </div>

                    <span className="font-bold text-slate-800 text-base group-hover/user:text-blue-600 transition-colors">
                      {link.user.username}
                    </span>
                    <span className="text-xs text-slate-400 truncate w-full text-center px-2">
                      {link.user.pvzAddress || 'ПВЗ не указан'}
                    </span>
                  </div>

                  {/* Блок товаров */}
                  <div className="w-full space-y-3 px-2">
                    {/* ЧТО ОТДАЕТ (Стрелка от этого блока идет вправо к следующему) */}
                    <ItemCard
                      item={link.givingItem}
                      label="Отдает"
                      color="slate"
                    />

                    {/* Стрелка вниз (визуальный разделитель) */}
                    <div className="flex justify-center text-slate-300">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 5v14M19 12l-7 7-7-7"/></svg>
                    </div>

                    {/* ЧТО ПОЛУЧАЕТ (Приходит от предыдущего или последнего) */}
                    {receivingItem && (
                      <ItemCard
                        item={receivingItem}
                        label={receivingLabel}
                        color="blue"
                      />
                    )}
                  </div>
                </div>

                {/* Стрелка передачи права/товара между участниками */}
                {!isLast && (
                  <div className="hidden md:flex flex-col items-center justify-center px-2 self-center mt-12">
                    <div className="flex items-center text-slate-300">
                      <div className="h-0.5 w-8 bg-slate-200"></div>
                      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-blue-500 -ml-1"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
                    </div>
                    <div className="text-[10px] text-slate-400 whitespace-nowrap mt-1 font-medium bg-white px-1 border border-slate-100 rounded shadow-sm">
                      передача: {link.givingItem.title}
                    </div>
                  </div>
                )}

                {/* Мобильная стрелка вниз */}
                {!isLast && (
                  <div className="md:hidden flex justify-center py-4 text-slate-300">
                     <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 5v14M19 12l-7 7-7-7"/></svg>
                  </div>
                )}

                {/* Замыкающая стрелка для последнего элемента (визуально показывает возврат к началу) */}
                {isLast && deal.chain.length > 1 && (
                   <div className="hidden md:flex absolute right-0 top-1/2 transform translate-x-full pl-4 text-slate-300 opacity-50 pointer-events-none">
                      {/* Здесь можно нарисовать дугу возврата, но для MVP достаточно текста */}
                      <div className="text-xs vertical-text rotate-90 origin-left whitespace-nowrap mt-20 text-slate-400">
                         ...замыкание на {deal.chain[0].user.username}
                      </div>
                   </div>
                )}
              </React.Fragment>
            );
          })}
        </div>

        {/* Индикатор замыкания цепи (для наглядности) */}
        {deal.chain.length > 1 && (
          <div className="mt-4 flex justify-center md:hidden">
             <div className="text-xs text-slate-400 bg-slate-50 px-3 py-1 rounded-full border border-slate-200">
                ↻ Цепочка замкнута: {deal.chain[deal.chain.length-1].givingItem.title} → {deal.chain[0].user.username}
             </div>
          </div>
        )}
      </div>

      {/* --- Ожидание --- */}
      {(deal.status === DealStatus.PENDING || deal.status === DealStatus.CONFIRMING) && (
         <div className="bg-yellow-50 border border-yellow-200 rounded-2xl p-6 text-center flex flex-col items-center">
            <div className="w-12 h-12 bg-yellow-100 rounded-full flex items-center justify-center text-2xl mb-3 animate-bounce">
              ⏳
            </div>
            <h3 className="text-lg font-bold text-yellow-800">Формирование цепочки</h3>
            <p className="text-yellow-600 text-sm mt-1 max-w-md">
              Участники рассматривают предложение. Как только все подтвердят готовность, начнется этап логистики.
            </p>
         </div>
      )}
    </div>
  );
};