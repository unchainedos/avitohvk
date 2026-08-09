// src/pages/item-page/ItemPage.tsx
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useState } from 'react';
import { itemStore } from '../../app/providers/ItemStore';
import { observer } from 'mobx-react-lite';
import { mockUsers } from '../../entities/user/api/userApi';
import { UserBadge } from '../../entities/user/ui/UserBadge';
import { useAuthStore } from '../../app/hooks/useAuthStore';
import { SuccessSticker } from '../../shared/ui/SuccessSticker';
import { ExchangeActionBtn } from '../../features/exchange-action/ui/ExchangeActionBtn';

export const ItemPage = observer(() => {
  const { id } = useParams();
  const navigate = useNavigate();
  const authStore = useAuthStore();
  const [mainImage, setMainImage] = useState('');
  const [showSticker, setShowSticker] = useState(false);
  const [stickerMessage, setStickerMessage] = useState('');

  const currentUser = authStore.user;

  const item = itemStore.getById(Number(id));
  if (item && !mainImage) {
    setMainImage(item.images?.[0] || item.imageUrl);
  }

  if (!item) return <div className="p-10 text-center">Товар не найден</div>;

  // Display holder (распорядитель) instead of author (владелец)
  const holder = mockUsers[item.holderId];
  const author = mockUsers[item.authorId];
  const isMyItem = currentUser && (item.holderId === currentUser.id || item.authorId === currentUser.id);
 
  const handleDealCreated = (dealId: string) => {
    setStickerMessage(`Сделка #${dealId} создана!`);
    setShowSticker(true);
    navigate(`/exchange/${dealId}`);
  };

  return (
    <div className="max-w-5xl mx-auto pb-20">
      <button onClick={() => navigate(-1)} className="mb-4 text-sm text-gray-500 hover:text-[#00AAFF] flex items-center gap-1 transition-colors">
        ← Назад к поиску
      </button>

      <div className="bg-white rounded-2xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-0">

          {/* Left column: Gallery */}
          <div className="bg-gray-50 p-6 flex flex-col gap-4 border-r border-gray-100">
            <div className="aspect-square rounded-xl overflow-hidden bg-white shadow-inner relative group">
              <img src={mainImage} alt={item.title} className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105" />
              {item.isLocked && (
                  <div className="absolute top-4 right-4 bg-red-500/90 text-white px-3 py-1 rounded-full text-xs font-bold backdrop-blur-sm">
                      Товар в сделке
                  </div>
              )}
            </div>

            {item.images && item.images.length > 1 && (
              <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-hide">
                {item.images.map((img, idx) => (
                  <button
                    key={idx}
                    onClick={() => setMainImage(img)}
                    className={`w-20 h-20 rounded-lg overflow-hidden border-2 shrink-0 transition-all ${mainImage === img ? 'border-[#00AAFF] opacity-100' : 'border-transparent opacity-60 hover:opacity-100'}`}
                  >
                    <img src={img} alt="" className="w-full h-full object-cover" />
                  </button>
                ))}
              </div>
            )}

            {/* Security block - hide for own items */}
            {!isMyItem && (
              <div className="mt-4 p-4 bg-yellow-50 border border-yellow-100 rounded-xl text-xs text-yellow-800 flex items-start gap-3">
                  <span className="text-lg">🔒</span>
                  <div>
                      <p className="font-bold mb-1">Безопасная сделка</p>
                      <p className="opacity-80 leading-relaxed">
                          Контакты продавца скрыты до момента подтверждения сделки и сдачи товара в ПВЗ.
                          Общение вне платформы запрещено правилами безопасности.
                      </p>
                  </div>
              </div>
            )}
          </div>

          {/* Right column: Info */}
          <div className="p-8 flex flex-col h-full">
            <div className="flex justify-between items-start mb-4">
                <span className="px-3 py-1 bg-gray-100 text-gray-600 text-xs font-bold rounded-full uppercase tracking-wide">{item.category}</span>
                <span className="text-sm text-gray-400 font-medium">ID: {item.id}</span>
            </div>

            <h1 className="text-3xl font-bold text-gray-900 mb-4 leading-tight">{item.title}</h1>

            <div className="prose prose-sm text-gray-600 mb-8 leading-relaxed whitespace-pre-line">
              {item.description}
            </div>

            {/* Owner wishes block */}
            {item.wishes && item.wishes.length > 0 && (
                <div className="bg-blue-50/50 rounded-2xl p-5 mb-8 border border-blue-100">
                    <h3 className="text-sm font-bold text-blue-900 mb-3 flex items-center gap-2">
                        <span>🎁</span> Что хочет владелец взамен:
                    </h3>
                    <div className="flex flex-wrap gap-2">
                        {item.wishes.map((wish, i) => (
                            <span key={i} className="px-3 py-1.5 bg-white border border-blue-200 text-blue-700 text-sm rounded-lg shadow-sm font-medium">
                                {wish}
                            </span>
                        ))}
                    </div>
                </div>
            )}

            <div className="mt-auto pt-8 border-t border-gray-100 flex flex-col gap-4">
                <div className="grid gap-4 sm:grid-cols-2">
                    <div className="flex flex-col">
                        <span className="text-xs text-gray-400 uppercase font-bold mb-2 tracking-wider">
                          {item.holderId !== item.authorId ? 'Распорядитель' : 'Владелец товара'}
                        </span>
                        {holder ? (
                            <Link to={`/user/${holder.id}`} className="flex items-center gap-3 bg-gray-50 p-2 pr-4 rounded-xl border border-gray-100 hover:border-[#00AAFF] transition-colors">
                                <UserBadge user={holder} />
                            </Link>
                        ) : (
                            <span className="text-gray-400 text-sm">Неизвестен</span>
                        )}
                    </div>
                    {item.holderId !== item.authorId && (
                      <div className="flex flex-col">
                        <span className="text-xs text-gray-400 uppercase font-bold mb-2 tracking-wider">
                          Владелец товара
                        </span>
                        {author ? (
                          <Link to={`/user/${author.id}`} className="flex items-center gap-3 bg-gray-50 p-2 pr-4 rounded-xl border border-gray-100 hover:border-[#00AAFF] transition-colors">
                            <UserBadge user={author} />
                          </Link>
                        ) : (
                          <span className="text-gray-400 text-sm">Неизвестен</span>
                        )}
                      </div>
                    )}
                </div>

                {/* Single smart action button */}
                {!item.isLocked && currentUser && (
                    <div className="flex flex-col gap-2 min-w-[200px]">
                        <ExchangeActionBtn
                          item={item}
                          onDealCreated={() => handleDealCreated('pending')}
                        />
                    </div>
                )}

                {item.isLocked && (
                  <div className="px-8 py-4 rounded-xl font-bold text-gray-400 bg-gray-100 border border-gray-200 cursor-not-allowed text-center min-w-[200px]">
                    Товар в сделке
                  </div>
                )}
            </div>
          </div>
        </div>
      </div>

      {/* Success Sticker for notifications */}
      <SuccessSticker
        message={stickerMessage}
        isVisible={showSticker}
        onClose={() => setShowSticker(false)}
        duration={4000}
      />
    </div>
  );
})