import type { AuthMode } from '../types/auth';

interface AuthTabsProps {
    mode: AuthMode;
    onModeChange: (mode: AuthMode) => void;
}

function AuthTabs({ mode, onModeChange }: AuthTabsProps) {
    return (
        <div className="auth-tabs">
            <div
                className={`auth-tab-indicator ${mode === 'login' ? 'login' : ''}`}
            />

            <button
                type="button"
                className={mode === 'register' ? 'active' : ''}
                onClick={() => onModeChange('register')}
                aria-pressed={mode === 'register'}
            >
                Регистрация
            </button>

            <button
                type="button"
                className={mode === 'login' ? 'active' : ''}
                onClick={() => onModeChange('login')}
                aria-pressed={mode === 'login'}
            >
                Вход
            </button>
        </div>
    );
}

export default AuthTabs;
