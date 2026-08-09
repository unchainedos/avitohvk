// src/features/decline-deal/ui/DeclineModal.tsx
import React from 'react';

interface DeclineModalProps {
  isOpen: boolean;
  onClose: () => void;
  reason: string;
  initiatorName: string;
  onFindNewDeal: () => void;
}

export const DeclineModal: React.FC<DeclineModalProps> = ({ 
  isOpen, 
  onClose, 
  reason, 
  initiatorName, 
  onFindNewDeal 
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full overflow-hidden transform transition-all scale-100">
        <div className="p-6 flex flex-col items-center text-center space-y-4">
          <div className="w-20 h-20 bg-red-100 rounded-full flex items-center justify-center text-4xl mb-2 shadow-inner">
            ⛓️‍💥
          </div>
          
          <h3 className="text-2xl font-bold text-slate-800">Цепочка распалась</h3>
          
          <p className="text-slate-600 leading-relaxed">
            Пользователь <span className="font-semibold text-slate-900">{initiatorName}</span> отказался от участия. 
            Цепочка обмена не может быть продолжена без этого звена.
          </p>
          
          <div className="bg-slate-50 p-4 rounded-xl text-sm text-slate-500 italic border border-slate-100 w-full relative">
            <span className="absolute -top-2 left-4 bg-slate-50 px-2 text-xs font-bold text-slate-400 uppercase">Причина</span>
            "{reason}"
          </div>

          <div className="w-full bg-amber-50 border border-amber-100 rounded-xl p-4 text-left text-sm text-amber-800 space-y-2 mt-2">
            <p className="font-bold flex items-center gap-2 text-amber-900">
              <span>⚠️</span> Последствия:
            </p>
            <ul className="list-disc list-inside space-y-1.5 ml-1 text-amber-700 marker:text-amber-400">
              <li>Все товары возвращаются владельцам (исключительное право отозвано).</li>
              <li>Залог виновника удержан в счет компенсации логистики.</li>
              <li>Рейтинг участника снижен за срыв сделки.</li>
            </ul>
          </div>

          <div className="flex gap-3 w-full pt-4">
            <button 
              onClick={onClose}
              className="flex-1 px-4 py-2.5 border border-slate-200 text-slate-600 rounded-xl font-medium hover:bg-slate-50 transition-colors"
            >
              Закрыть
            </button>
            <button 
              onClick={onFindNewDeal}
              className="flex-1 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium shadow-md shadow-blue-200 transition-all hover:-translate-y-0.5"
            >
              Найти новый обмен
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};