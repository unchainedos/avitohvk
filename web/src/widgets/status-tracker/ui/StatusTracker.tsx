// src/widgets/status-tracker/ui/StatusTracker.tsx
import React from 'react';
// ИСПРАВЛЕНО: путь к твоим типам
import { LogisticsStatus } from '../../../shared/api/types';

interface StatusTrackerProps {
  currentStatus: LogisticsStatus;
  onDropOff: () => void;
  isCurrentUserTurn: boolean;
}

const STEPS = [
  { key: LogisticsStatus.PENDING_DROP_OFF, label: 'Ожидает сдачи', icon: '📦', desc: 'Отнесите в ПВЗ' },
  { key: LogisticsStatus.DROPPED_OFF, label: 'Принято', icon: '🏢', desc: 'Проверено сотрудником' },
  { key: LogisticsStatus.IN_TRANSIT, label: 'В пути', icon: '🚚', desc: 'Едет в другой город' },
  { key: LogisticsStatus.DELIVERED_TO_PVZ, label: 'Доставлено', icon: '📍', desc: 'Ждет получателя' },
  { key: LogisticsStatus.COMPLETED, label: 'Получено', icon: '✅', desc: 'Сделка завершена' },
];

export const StatusTracker: React.FC<StatusTrackerProps> = ({ currentStatus, onDropOff, isCurrentUserTurn }) => {
  const currentIndex = STEPS.findIndex(s => s.key === currentStatus);

  return (
    <div className="w-full bg-slate-50 border-t border-slate-200 p-6 md:p-8 rounded-b-2xl">
      <div className="flex justify-between items-center mb-6">
        <h4 className="text-sm font-bold text-slate-500 uppercase tracking-wider">
          Трекинг обмена
        </h4>
        {currentStatus !== LogisticsStatus.NONE && (
          <span className="text-xs font-medium bg-blue-100 text-blue-700 px-2 py-1 rounded-full">
            Этап {currentIndex + 1} из {STEPS.length}
          </span>
        )}
      </div>
      
      <div className="relative">
        <div className="absolute top-5 left-0 w-full h-1 bg-slate-200 -z-10 rounded-full hidden md:block" />
        
        <div 
          className="absolute top-5 left-0 h-1 bg-blue-500 -z-10 rounded-full transition-all duration-1000 ease-out hidden md:block"
          style={{ width: currentIndex >= 0 ? `${(currentIndex / (STEPS.length - 1)) * 100}%` : '0%' }}
        />

        <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6 md:gap-0">
          {STEPS.map((step, index) => {
            const isCompleted = index <= currentIndex;
            const isCurrent = index === currentIndex;

            return (
              <div key={step.key} className="flex md:flex-col items-center md:items-center w-full md:w-auto group relative">
                <div className="flex items-center md:flex-col">
                  <div 
                    className={`
                      w-10 h-10 rounded-full flex items-center justify-center text-lg border-2 transition-all duration-300 z-10 bg-white
                      ${isCompleted ? 'border-blue-500 text-blue-600 shadow-md' : 'border-slate-300 text-slate-300'}
                      ${isCurrent ? 'ring-4 ring-blue-100 scale-110 border-blue-600' : ''}
                    `}
                  >
                    {isCompleted && !isCurrent ? '✓' : step.icon}
                  </div>
                  
                  <div className="ml-3 md:ml-0 md:mt-3 md:text-center w-full md:w-32">
                    <p className={`text-sm font-bold ${isCompleted ? 'text-slate-800' : 'text-slate-400'}`}>
                      {step.label}
                    </p>
                    <p className="text-xs text-slate-400 hidden md:block mt-1">
                      {step.desc}
                    </p>
                  </div>
                </div>

                {index < STEPS.length - 1 && (
                  <div className="md:hidden absolute left-5 top-10 w-0.5 h-full bg-slate-200 -z-10" 
                       style={{ height: 'calc(100% + 1.5rem)' }} />
                )}
                {index < currentIndex && (
                   <div className="md:hidden absolute left-5 top-10 w-0.5 h-full bg-blue-500 -z-10" 
                        style={{ height: 'calc(100% + 1.5rem)' }} />
                )}
              </div>
            );
          })}
        </div>
      </div>

      <div className="mt-8 flex justify-center h-12">
        {isCurrentUserTurn && currentStatus === LogisticsStatus.PENDING_DROP_OFF && (
          <button 
            onClick={onDropOff}
            className="px-8 py-3 bg-green-600 hover:bg-green-700 text-white rounded-xl font-bold shadow-lg shadow-green-200 transform transition hover:-translate-y-0.5 flex items-center gap-2"
          >
            <span>📦</span> Я сдал товар в ПВЗ
          </button>
        )}
        
        {isCurrentUserTurn && currentStatus === LogisticsStatus.DELIVERED_TO_PVZ && (
          <button className="px-8 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-bold shadow-lg shadow-blue-200 transition">
            Забрать товар
          </button>
        )}

        {!isCurrentUserTurn && currentStatus !== LogisticsStatus.NONE && currentStatus !== LogisticsStatus.COMPLETED && (
          <div className="text-slate-400 text-sm flex items-center gap-2">
            <span className="w-2 h-2 bg-slate-300 rounded-full animate-pulse"></span>
            Ожидание действий другого участника...
          </div>
        )}
      </div>
    </div>
  );
};