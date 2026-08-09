// src/pages/user-profile-page/UserProfilePage.tsx
import React, { useEffect, useState } from "react";
import { observer } from 'mobx-react-lite';
import { useParams, useNavigate, Link } from "react-router-dom";
import type { IUser } from "../../shared/api/types";
import { authApi } from "../../shared/api/authApi";
import { itemStore } from "../../app/providers/ItemStore";
import { mockUsers } from "../../entities/user/api/userApi";

export const UserProfilePage: React.FC = observer(() => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [user, setUser] = useState<IUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadUserData = async () => {
      setLoading(true);
      try {
        if (id) {
          const userId = Number(id);
          const userData = await authApi.getUserById(userId);
          if (userData) {
            setUser(userData);
          } else {
            setUser(null);
          }
        }
      } catch (error) {
        console.error("Failed to load user data:", error);
        setUser(null);
      } finally {
        setLoading(false);
      }
    };

    loadUserData();
  }, [id]);

  // derive visible items directly so updates in itemStore reflect immediately
  const items = user ? itemStore.all.filter(
    (item) => item.authorId === user.id || (item.holderId === user.id && item.authorId !== user.id),
  ) : [];


  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh]">
        <h2 className="text-2xl font-bold text-slate-800 mb-4">
          Пользователь не найден
        </h2>
        <button
          onClick={() => navigate(-1)}
          className="px-6 py-3 bg-[#00AAFF] text-white rounded-lg hover:bg-[#0095E0] transition font-medium"
        >
          Назад
        </button>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto p-4 md:p-8">
      {/* Profile Header */}
      <div className="bg-white rounded-2xl shadow-sm border border-slate-100 p-6 mb-8 flex flex-col md:flex-row items-center md:items-start gap-6">
        <Link
          to={`/user/${user.id}`}
          className="w-24 h-24 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white text-3xl font-bold shadow-lg flex-shrink-0 hover:opacity-90 transition-opacity overflow-hidden"
        >
          {user.avatarUrl ? (
            <img
              src={user.avatarUrl}
              alt=""
              className="w-full h-full object-cover"
            />
          ) : (
            user.username.charAt(0).toUpperCase()
          )}
        </Link>

        <div className="flex-1 text-center md:text-left">
          <h1 className="text-3xl font-bold text-slate-800 mb-2">
            {user.username}
          </h1>

          <div className="flex flex-wrap justify-center md:justify-start gap-4 text-sm text-slate-500 mb-4">
            <span className="flex items-center gap-1 bg-yellow-50 text-yellow-700 px-3 py-1 rounded-full font-medium">
              ★ {user.rating} Рейтинг
            </span>
            <span className="flex items-center gap-1 bg-slate-100 px-3 py-1 rounded-full">
              📍 {user.pvzAddress || "ПВЗ не указан"}
            </span>
            {user.declineCount > 0 && (
              <span className="flex items-center gap-1 bg-red-50 text-red-600 px-3 py-1 rounded-full">
                ⚠️ Отказов: {user.declineCount}
              </span>
            )}
          </div>

          <p className="text-slate-600 max-w-2xl">
            Участник системы многостороннего обмена. Ценит честность и
            пунктуальность.
          </p>
        </div>

        <button
          onClick={() => navigate(-1)}
          className="px-4 py-2 border border-slate-200 rounded-lg text-slate-600 hover:bg-slate-50 transition self-start"
        >
          ← Назад
        </button>
      </div>

      {/* User's Items */}
      <div>
        <h2 className="text-xl font-bold text-slate-800 mb-6 flex items-center gap-2">
          <span>📦</span> Товары пользователя ({items.length})
        </h2>

        {items.length === 0 ? (
          <div className="text-center py-12 bg-slate-50 rounded-xl border border-dashed border-slate-300">
            <p className="text-slate-500">
              У пользователя пока нет товаров для обмена
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
              {items.map((item) => {
              const ownerUser = mockUsers[item.authorId] || { id: 0, username: 'Unknown', rating: 0, declineCount: 0 };
              const hasExclusiveRight =
                item.holderId === user?.id && item.authorId !== user?.id;
              const isOwner = item.authorId === user?.id;
 
              return (                <div
                  key={item.id}
                  className={`bg-white rounded-xl border overflow-hidden hover:shadow-md transition-shadow group ${hasExclusiveRight ? "border-purple-200 ring-1 ring-purple-100" : "border-slate-200"}`}
                >
                  <div className="aspect-video bg-slate-100 relative overflow-hidden">
                    <img
                      src={item.imageUrl}
                      alt={item.title}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    {(item.isLocked || hasExclusiveRight) && (
                      <div className={`absolute top-2 left-2 text-white text-xs px-2 py-1 rounded font-bold shadow-sm flex items-center gap-1 ${item.isLocked ? 'bg-orange-500' : 'bg-purple-600'}`}>
                        {item.isLocked ? '🔒 В сделке' : '⚡ Исключительное право'}
                      </div>
                    )}
                  </div>

                  <div className="p-4">
                    <div className="flex justify-between items-start mb-2">
                      <h3 className="font-bold text-slate-800 truncate pr-2">
                        {item.title}
                      </h3>
                      <span className="text-xs bg-slate-100 text-slate-500 px-2 py-0.5 rounded whitespace-nowrap">
                        {item.quantity} {item.unit}
                      </span>
                    </div>
                    <p className="text-sm text-slate-500 line-clamp-2 mb-3 h-10">
                      {item.description}
                    </p>

                    {item.wishes && item.wishes.length > 0 && (
                      <div className="mb-3 pt-3 border-t border-slate-100">
                        <p className="text-xs text-slate-400 mb-2">
                          Хочет взамен:
                        </p>
                        <div className="flex flex-wrap gap-1">
                          {item.wishes.slice(0, 3).map((wish, i) => (
                            <span
                              key={i}
                              className="text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded"
                            >
                              {wish}
                            </span>
                          ))}
                          {item.wishes.length > 3 && (
                            <span className="text-xs text-slate-400">
                              +{item.wishes.length - 3}
                            </span>
                          )}
                        </div>
                      </div>
                    )}

                    <div className="flex items-center justify-between text-xs text-slate-400 pt-3 border-t border-slate-100">
                      <span>{item.category}</span>
                      {hasExclusiveRight ? (
                        <span className="text-purple-600 font-medium">
                          Владелец: {ownerUser.username}
                        </span>
                      ) : isOwner ? (
                        <span>ID: {item.id}</span>
                      ) : (
                        <span className="text-purple-600 font-medium">
                          Владелец: другой пользователь
                        </span>
                      )}
                    </div>

                    <Link
                      to={`/item/${item.id}`}
                      className="mt-3 block w-full text-center py-2 bg-slate-50 hover:bg-[#00AAFF] hover:text-white text-slate-600 rounded-lg text-sm font-medium transition-colors"
                    >
                      Подробнее
                    </Link>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
})
