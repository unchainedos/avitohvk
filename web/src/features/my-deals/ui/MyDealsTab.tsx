// src/features/my-deals/ui/MyDealsTab.tsx
import { useState, useMemo } from "react";
import { observer } from 'mobx-react-lite';
import { useNavigate } from "react-router-dom";
import { useStore } from "../../../app/providers/StoreProvider";
import { dealStore } from "../../../app/providers/DealStore";
import { DealStatus, ChainLinkStatus } from "../../../shared/api/types";
import { DealCard } from "../../../widgets/deal-card/ui/DealCard";

interface MyDealsTabProps {
  currentUserId: number;
}

export const MyDealsTab = observer(({ currentUserId }: MyDealsTabProps) => {
  const navigate = useNavigate();
  const store = useStore();
  const [filter, setFilter] = useState<"active" | "pending" | "completed" | "cancelled" | "all">("all");

  // Get deals from dealStore
  // derive deals directly from dealStore so MobX reactivity triggers updates
  const myDeals = dealStore.getDealsForUser(currentUserId);

  // Apply status filter
  const filteredDeals = useMemo(() => {
    if (filter === "all") return myDeals;
    return myDeals.filter((deal) => {
      if (filter === "active") return deal.status === DealStatus.ACTIVE;
      if (filter === "pending") return deal.status === DealStatus.PENDING;
      if (filter === "completed") return deal.status === DealStatus.COMPLETED;
      if (filter === "cancelled") return deal.status === DealStatus.CANCELLED;
      return true;
    });
  }, [myDeals, filter]);

  const handleConfirm = (dealId: string) => {
    dealStore.confirmDeal(dealId);
  };

  const handleCancel = (dealId: string, reason?: string) => {
    dealStore.cancelDeal(dealId, reason);
  };

  const getStatusBadge = (status: DealStatus) => {
    switch (status) {
      case DealStatus.PENDING:
        return (
          <span className="px-2.5 py-1 bg-yellow-100 text-yellow-700 text-xs font-bold rounded-full uppercase border border-yellow-200">
            Ожидает
          </span>
        );
      case DealStatus.ACTIVE:
        return (
          <span className="px-2.5 py-1 bg-green-100 text-green-700 text-xs font-bold rounded-full uppercase border border-green-200">
            Активен
          </span>
        );
      case DealStatus.CANCELLED:
        return (
          <span className="px-2.5 py-1 bg-red-100 text-red-700 text-xs font-bold rounded-full uppercase border border-red-200">
            Отменен
          </span>
        );
      case DealStatus.COMPLETED:
        return (
          <span className="px-2.5 py-1 bg-blue-100 text-blue-700 text-xs font-bold rounded-full uppercase border border-blue-200">
            Завершен
          </span>
        );
      default:
        return null;
    }
  };

  return (
    <div className="space-y-4">
      {/* Header and filter */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-2">
        <h2 className="text-lg font-bold text-gray-900">Мои сделки</h2>

        <div className="flex gap-2 flex-wrap">
          {(["all", "pending", "active", "completed", "cancelled"] as const).map((status) => (
            <button
              key={status}
              onClick={() => setFilter(status)}
              className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                filter === status
                  ? "bg-[#00AAFF] text-white shadow-sm"
                  : "bg-gray-100 text-gray-600 hover:bg-gray-200"
              }`}
            >
              {status === "all" ? "Все" :
               status === "pending" ? "Ожидают" :
               status === "active" ? "Активные" :
               status === "completed" ? "Завершенные" :
               "Отмененные"}
            </button>
          ))}
        </div>
      </div>

      {/* Deals list */}
      {filteredDeals.length > 0 ? (
        <div className="space-y-4">
          {filteredDeals.map((deal) => (
            <DealCard
              key={deal.id}
              deal={deal}
              currentUserId={currentUserId}
              onConfirm={handleConfirm}
              onCancel={handleCancel}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-16 text-gray-400 flex flex-col items-center gap-4">
          <div className="w-16 h-16 bg-gray-50 rounded-full flex items-center justify-center text-2xl">
            📦
          </div>
          <p>
            {filter === "all" && "У вас пока нет сделок"}
            {filter === "active" && "У вас нет активных обменов"}
            {filter === "pending" && "У вас нет ожидающих подтверждений обменов"}
            {filter === "completed" && "У вас нет завершенных обменов"}
            {filter === "cancelled" && "У вас нет отмененных обменов"}
          </p>
          {filter === "all" && (
            <button
              onClick={() => navigate("/")}
              className="text-[#00AAFF] text-sm font-medium hover:underline"
            >
              Создать новый обмен
            </button>
          )}
        </div>
      )}
    </div>
  );
})