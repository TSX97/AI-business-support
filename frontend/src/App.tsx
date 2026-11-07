
import './App.css';
import { useState, type FormEvent } from 'react';

type AuthMode = 'register' | 'login';

interface ApiResponse {
    message?: string;
}

function App() {
    const [mode, setMode] = useState<AuthMode>('register');

    const [name, setName] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');

    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();

        setError('');
        setSuccess('');
        setLoading(true);

        try {
            const endpoint =
                mode === 'register'
                    ? '/api/auth/register'
                    : '/api/auth/login';

            const body =
                mode === 'register'
                    ? { name, email, password }
                    : { email, password };

            const response = await fetch(endpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(body),
            });

            const data = await response.json().catch(() => ({})) as ApiResponse;

            if (!response.ok) {
                throw new Error(data.message || 'Произошла ошибка');
            }

            setSuccess(
                mode === 'register'
                    ? 'Аккаунт успешно создан!'
                    : 'Вы успешно вошли в аккаунт!',
            );

            setPassword('');

            if (mode === 'register') {
                setName('');
                setEmail('');
            }
        } catch (err) {
            setError(
                err instanceof Error ? err.message : 'Что-то пошло не так',
            );
        } finally {
            setLoading(false);
        }
    };

    const changeMode = (newMode: AuthMode) => {
        if (loading) return;

        setMode(newMode);
        setError('');
        setSuccess('');
    };

    return (
        <main className="page">
            <header className="header">
                <div className="brand">
                    <div className="brand-mark">AI</div>
                    <div className="brand-name">
                        <span>AI</span>
                        <span>Бизнес Помощник</span>
                    </div>
                </div>

                <span className="header-caption">
                    Умные решения для бизнеса
                </span>
            </header>

            <section className="registration-wrapper">
                <form
                    className={`registration-card ${mode === 'login' ? 'login-mode' : 'register-mode'}`}
                    onSubmit={handleSubmit}
                >
                    <div className="auth-tabs">
                        <div
                            className={`auth-tab-indicator ${
                                mode === 'login' ? 'login' : ''
                            }`}
                        />

                        <button
                            type="button"
                            className={mode === 'register' ? 'active' : ''}
                            onClick={() => changeMode('register')}
                            aria-pressed={mode === 'register'}
                        >
                            Регистрация
                        </button>

                        <button
                            type="button"
                            className={mode === 'login' ? 'active' : ''}
                            onClick={() => changeMode('login')}
                            aria-pressed={mode === 'login'}
                        >
                            Вход
                        </button>
                    </div>

                    <div className="card-heading">
                        <span className="eyebrow">
                            {mode === 'register'
                                ? 'ДОБРО ПОЖАЛОВАТЬ'
                                : 'С ВОЗВРАЩЕНИЕМ'}
                        </span>

                        <h1>
                            {mode === 'register'
                                ? 'Создать аккаунт'
                                : 'Войти в аккаунт'}
                        </h1>

                        <p>
                            {mode === 'register'
                                ? 'Заполните данные, чтобы начать работу'
                                : 'Введите данные своего аккаунта'}
                        </p>
                    </div>

                    <div className={`name-field ${mode === 'login' ? 'hidden' : ''}`}>
                        <div className="input-group">
                            <label htmlFor="name">Ваше имя</label>
                            <input
                                id="name"
                                type="text"
                                placeholder="Например, Иван"
                                autoComplete="name"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                required={mode === 'register'}
                                tabIndex={mode === 'register' ? 0 : -1}
                            />
                        </div>
                    </div>

                    <div className="input-group">
                        <label htmlFor="email">Электронная почта</label>
                        <input
                            id="email"
                            type="email"
                            placeholder="name@example.com"
                            autoComplete="email"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            required
                        />
                    </div>

                    <div className="input-group">
                        <label htmlFor="password">Пароль</label>
                        <input
                            id="password"
                            type="password"
                            placeholder={
                                mode === 'register'
                                    ? 'Минимум 8 символов'
                                    : 'Ваш пароль'
                            }
                            autoComplete={
                                mode === 'register'
                                    ? 'new-password'
                                    : 'current-password'
                            }
                            minLength={mode === 'register' ? 8 : undefined}
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            required
                        />
                    </div>

                    {error && (
                        <p className="message error" role="alert">
                            {error}
                        </p>
                    )}

                    {success && (
                        <p className="message success" role="status">
                            {success}
                        </p>
                    )}

                    <button
                        className="submit-button"
                        type="submit"
                        disabled={loading}
                    >
                        {loading
                            ? 'Подождите...'
                            : mode === 'register'
                                ? 'Зарегистрироваться'
                                : 'Войти'}
                    </button>

                    <p className="login-hint">
                        {mode === 'register'
                            ? 'Уже есть аккаунт?'
                            : 'Ещё нет аккаунта?'}
                        {' '}

                        <a
                            href="#"
                            onClick={(e) => {
                                e.preventDefault();
                                changeMode(
                                    mode === 'register' ? 'login' : 'register',
                                );
                            }}
                        >
                            {mode === 'register'
                                ? 'Войти'
                                : 'Создать аккаунт'}
                        </a>
                    </p>
                </form>
            </section>

            <footer className="footer">
                AI Бизнес Помощник — ваш интеллектуальный партнёр
            </footer>
        </main>
    );
}

export default App;
