// src/shared/ui/SuccessSticker.tsx
import React, { useEffect } from 'react';

interface SuccessStickerProps {
  message: string;
  isVisible: boolean;
  onClose: () => void;
  duration?: number;
}

export const SuccessSticker: React.FC<SuccessStickerProps> = ({
  message,
  isVisible,
  onClose,
  duration = 2500,
}) => {
  useEffect(() => {
    if (isVisible) {
      const timer = setTimeout(() => {
        onClose();
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [isVisible, onClose, duration]);

  if (!isVisible) return null;

  return (
    <div className="fixed top-4 right-4 z-[1000] animate-in slide-in-from-right fade-out duration-500">
      <div className="bg-white rounded-xl shadow-lg p-4 border-l-4 border-green-500 flex items-center gap-3 max-w-sm transition-opacity duration-500">
        {/* Иконка успеха */}
        <div className="w-8 h-8 bg-gradient-to-br from-green-400 to-green-500 rounded-full flex items-center justify-center flex-shrink-0">
          <svg
            className="w-5 h-5 text-white"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={3}
              d="M5 13l4 4L19 7"
            />
          </svg>
        </div>

        {/* Текст сообщения */}
        <p className="text-sm font-medium text-gray-800">
          {message}
        </p>
      </div>
    </div>
  );
};