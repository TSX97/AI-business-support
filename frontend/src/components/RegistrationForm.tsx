import { useState, type FormEvent } from 'react';
import AuthFields from './AuthFields';
import AuthMessages from './AuthMessages';
import AuthTabs from './AuthTabs';
import { submitAuth } from '../services/auth';
import type { AuthMode } from '../types/auth';

function RegistrationForm() {
    const [mode, setMode] = useState<AuthMode>('register');
    const [name, setName] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    const changeMode = (newMode: AuthMode) => {
        if (loading) return;

        setMode(newMode);
        setError('');
        setSuccess('');
    };

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        setError('');
        setSuccess('');
        setLoading(true);

        try {
            await submitAuth(mode, { name, email, password });

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

    const isRegistering = mode === 'register';

    return (
        <form
            className={`registration-card ${isRegistering ? 'register-mode' : 'login-mode'}`}
            onSubmit={handleSubmit}
        >
            <AuthTabs mode={mode} onModeChange={changeMode} />

            <div className="card-heading">
                <span className="eyebrow">
                    {isRegistering ? 'ДОБРО ПОЖАЛОВАТЬ' : 'С ВОЗВРАЩЕНИЕМ'}
                </span>
                <h1>{isRegistering ? 'Создать аккаунт' : 'Войти в аккаунт'}</h1>
                <p>
                    {isRegistering
                        ? 'Заполните данные, чтобы начать работу'
                        : 'Введите данные своего аккаунта'}
                </p>
            </div>

            <AuthFields
                mode={mode}
                name={name}
                email={email}
                password={password}
                onNameChange={setName}
                onEmailChange={setEmail}
                onPasswordChange={setPassword}
            />

            <AuthMessages error={error} success={success} />

            <button className="submit-button" type="submit" disabled={loading}>
                {loading
                    ? 'Подождите...'
                    : isRegistering
                        ? 'Зарегистрироваться'
                        : 'Войти'}
            </button>

            <p className="login-hint">
                {isRegistering ? 'Уже есть аккаунт?' : 'Ещё нет аккаунта?'}{' '}
                <a
                    href="#"
                    onClick={(event) => {
                        event.preventDefault();
                        changeMode(isRegistering ? 'login' : 'register');
                    }}
                >
                    {isRegistering ? 'Войти' : 'Создать аккаунт'}
                </a>
            </p>
        </form>
    );
}

export default RegistrationForm;
