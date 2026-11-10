import type { AuthMode } from '../types/auth';

interface ApiResponse {
    message?: string;
}

interface AuthCredentials {
    name: string;
    email: string;
    password: string;
}

export async function submitAuth(
    mode: AuthMode,
    { name, email, password }: AuthCredentials,
): Promise<void> {
    const endpoint = mode === 'register' ? '/api/auth/register' : '/api/auth/login';
    const body = mode === 'register' ? { name, email, password } : { email, password };

    const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
    });

    const data = (await response.json().catch(() => ({}))) as ApiResponse;

    if (!response.ok) {
        throw new Error(data.message || 'Произошла ошибка');
    }
}
