// src/features/create-item/ui/CreateItemForm.tsx
import { useState, useEffect } from 'react';
import type { IItem } from '../../../shared/api/types';
import { itemApi } from '../../../entities/item/api/itemApi';
import { authApi } from '../../../shared/api/authApi';
import { useAuthStore } from '../../../app/hooks/useAuthStore';

interface CreateItemFormProps {
  onSuccess: () => void;
  onCancel: () => void;
}

export const CreateItemForm = ({ onSuccess, onCancel }: CreateItemFormProps) => {
  const authStore = useAuthStore();
  const [formData, setFormData] = useState<Partial<IItem>>({
    title: '',
    description: '',
    category: '',
    quantity: 1,
    unit: 'шт',
    imageUrl: 'https://placehold.co/400x300/e2e8f0/64748b?text=No+Image', // Дефолтная заглушка
    wishes: [],
  });
  const [isLoading, setIsLoading] = useState(false);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [userWishes, setUserWishes] = useState<string[]>([]);
  const [selectedWishes, setSelectedWishes] = useState<string[]>([]);
  const [customWish, setCustomWish] = useState('');

  // Load user wishes on mount
  useEffect(() => {
    if (authStore.user?.id) {
      authApi.getUserWishes(authStore.user.id).then(setUserWishes);
    }
  }, [authStore.user?.id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      // If image file is selected, create a local URL for it
      let finalImageUrl = formData.imageUrl;
      if (imageFile) {
        finalImageUrl = URL.createObjectURL(imageFile);
      }

      // Combine selected wishes and custom wish
      const allWishes = [...selectedWishes];
      if (customWish.trim() && !allWishes.includes(customWish.trim())) {
        allWishes.push(customWish.trim());
        // Add to user's wishes automatically
        await authApi.addWish(authStore.user!.id, customWish.trim());
      }

      await itemApi.createItem({ ...formData, imageUrl: finalImageUrl, wishes: allWishes });
      onSuccess();
    } catch (error) {
      console.error('Failed to create item', error);
    } finally {
      setIsLoading(false);
    }
  };

  const toggleWish = (wish: string) => {
    setSelectedWishes(prev =>
      prev.includes(wish) ? prev.filter(w => w !== wish) : [...prev, wish]
    );
  };

  return (
    <div className="bg-white p-6 rounded-xl shadow-lg border border-gray-100 max-w-md w-full max-h-[90vh] overflow-y-auto">
      <h2 className="text-xl font-bold text-gray-900 mb-4">Добавить вещь</h2>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Название</label>
          <input
            required
            type="text"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#00AAFF] focus:border-transparent outline-none"
            value={formData.title}
            onChange={e => setFormData({...formData, title: e.target.value})}
            placeholder="Например: Велосипед"
          />
        </div>

        {/* Новое поле для загрузки фото с устройства */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Фото товара</label>
          <div className="flex items-center gap-4">
            <div className="w-20 h-20 rounded-lg bg-gray-100 overflow-hidden border border-gray-300 flex items-center justify-center">
              {imageFile ? (
                <img src={URL.createObjectURL(imageFile)} alt="Preview" className="w-full h-full object-cover" />
              ) : (
                <span className="text-xs text-gray-400">Нет фото</span>
              )}
            </div>
            <label className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-200 transition-colors cursor-pointer">
              Выбрать файл
              <input
                type="file"
                accept="image/*"
                className="hidden"
                onChange={e => {
                  const file = e.target.files?.[0];
                  if (file) {
                    setImageFile(file);
                  }
                }}
              />
            </label>
          </div>
          <p className="text-xs text-gray-400 mt-1">Или вставьте ссылку ниже</p>
        </div>

        {/* Поле для ссылки на фото (альтернатива) */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Ссылка на фото</label>
          <input
            type="url"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#00AAFF] focus:border-transparent outline-none text-sm"
            value={formData.imageUrl}
            onChange={e => setFormData({...formData, imageUrl: e.target.value})}
            placeholder="https://..."
          />
          <p className="text-xs text-gray-400 mt-1">Вставьте прямую ссылку на изображение</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
          <textarea
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#00AAFF] focus:border-transparent outline-none h-20 resize-none"
            value={formData.description}
            onChange={e => setFormData({...formData, description: e.target.value})}
            placeholder="Состояние, комплектация..."
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Категория</label>
            <input
              required
              type="text"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#00AAFF] outline-none"
              value={formData.category}
              onChange={e => setFormData({...formData, category: e.target.value})}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Количество</label>
            <input
              required
              type="number"
              min="1"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#00AAFF] outline-none"
              value={formData.quantity}
              onChange={e => setFormData({...formData, quantity: Number(e.target.value)})}
            />
          </div>
        </div>

        {/* Wishes section */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Что хочу взамен</label>

          {/* Selected wishes tags */}
          {selectedWishes.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {selectedWishes.map(wish => (
                <span
                  key={wish}
                  className="inline-flex items-center gap-1 px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm"
                >
                  {wish}
                  <button
                    type="button"
                    onClick={() => toggleWish(wish)}
                    className="hover:text-red-500"
                  >
                    ✕
                  </button>
                </span>
              ))}
            </div>
          )}

          {/* User's existing wishes */}
          {userWishes.length > 0 && (
            <div className="mb-3">
              <p className="text-xs text-gray-500 mb-2">Выберите из ваших пожеланий:</p>
              <div className="flex flex-wrap gap-2 max-h-32 overflow-y-auto p-2 border border-gray-200 rounded-lg">
                {userWishes.map(wish => (
                  <button
                    key={wish}
                    type="button"
                    onClick={() => toggleWish(wish)}
                    className={`px-3 py-1 rounded-full text-sm transition-colors ${
                      selectedWishes.includes(wish)
                        ? 'bg-[#00AAFF] text-white'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                    }`}
                  >
                    {wish}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Custom wish input */}
          <div className="mt-3">
            <div className="flex gap-2">
              <input
                type="text"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#00AAFF] outline-none text-sm"
                value={customWish}
                onChange={e => setCustomWish(e.target.value)}
                placeholder="Или впишите своё пожелание..."
                onKeyDown={e => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    if (customWish.trim()) {
                      toggleWish(customWish.trim());
                      setCustomWish('');
                    }
                  }
                }}
              />
              <button
                type="button"
                onClick={() => {
                  if (customWish.trim()) {
                    toggleWish(customWish.trim());
                    setCustomWish('');
                  }
                }}
                className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-200 transition-colors"
              >
                Добавить
              </button>
            </div>
            <p className="text-xs text-gray-400 mt-1">Новые пожелания автоматически добавятся к вашим</p>
          </div>
        </div>

        <div className="flex gap-3 pt-2 sticky bottom-0 bg-white pt-4 border-t">
          <button
            type="button"
            onClick={onCancel}
            className="flex-1 py-2 border border-gray-300 text-gray-700 rounded-lg font-medium hover:bg-gray-50 transition-colors"
          >
            Отмена
          </button>
          <button
            type="submit"
            disabled={isLoading}
            className="flex-1 py-2 bg-[#00AAFF] text-white rounded-lg font-medium hover:bg-[#0095E0] transition-colors disabled:opacity-50"
          >
            {isLoading ? 'Сохранение...' : 'Добавить'}
          </button>
        </div>
      </form>
    </div>
  );
};