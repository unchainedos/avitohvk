// src/pages/auth-page/AuthPage.tsx
import { AuthForm } from '../../features/auth/ui/AuthForm';
import { Link, useNavigate, useLocation } from 'react-router-dom';

export const AuthPage = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const handleAuthSuccess = () => {
    // После успешной авторизации/регистрации перенаправляем на главную
    const from = location.state?.from || '/';
    navigate(from, { replace: true });
  };

  return (
    <div className="min-h-[calc(100vh-80px)] flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md space-y-8">
        {/* Header */}
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900">
            Вход и регистрация
          </h1>
          <p className="mt-2 text-sm text-gray-600">
            Войдите в свой аккаунт или создайте новый
          </p>
        </div>

        {/* Auth Form */}
        <AuthForm onSuccess={handleAuthSuccess} />

        {/* Back to home link */}
        <div className="text-center mt-6">
          <Link
            to="/"
            className="text-sm font-medium text-[#00AAFF] hover:text-[#0095E0] transition-colors"
          >
            ← Вернуться на главную
          </Link>
        </div>
      </div>
    </div>
  );
};