import type { AuthMode } from '../types/auth';

interface AuthFieldsProps {
    mode: AuthMode;
    name: string;
    email: string;
    password: string;
    onNameChange: (value: string) => void;
    onEmailChange: (value: string) => void;
    onPasswordChange: (value: string) => void;
}

function AuthFields({
    mode,
    name,
    email,
    password,
    onNameChange,
    onEmailChange,
    onPasswordChange,
}: AuthFieldsProps) {
    const isRegistering = mode === 'register';

    return (
        <>
            <div className={`name-field ${isRegistering ? '' : 'hidden'}`}>
                <div className="input-group">
                    <label htmlFor="name">Ваше имя</label>
                    <input
                        id="name"
                        type="text"
                        placeholder="Например, Иван"
                        autoComplete="name"
                        value={name}
                        onChange={(event) => onNameChange(event.target.value)}
                        required={isRegistering}
                        tabIndex={isRegistering ? 0 : -1}
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
                    onChange={(event) => onEmailChange(event.target.value)}
                    required
                />
            </div>

            <div className="input-group">
                <label htmlFor="password">Пароль</label>
                <input
                    id="password"
                    type="password"
                    placeholder={isRegistering ? 'Минимум 8 символов' : 'Ваш пароль'}
                    autoComplete={isRegistering ? 'new-password' : 'current-password'}
                    minLength={isRegistering ? 8 : undefined}
                    value={password}
                    onChange={(event) => onPasswordChange(event.target.value)}
                    required
                />
            </div>
        </>
    );
}

export default AuthFields;
