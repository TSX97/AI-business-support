import type { ChatReply } from '../types/chat';

interface ChatApiResponse {
    chat_id?: string | number;
    reply?: {
        content?: string;
    };
    message?: string;
}

async function postMessage(
    endpoint: string,
    content: string,
): Promise<ChatReply> {
    let response: Response;

    try {
        response = await fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ content }),
        });
    } catch {
        throw new Error('Не удалось связаться с сервером. Проверьте подключение и повторите попытку.');
    }

    if (response.status === 404) {
        throw new Error('AI-чат пока не подключён к серверу. Попробуйте позже.');
    }

    let data: ChatApiResponse;
    try {
        data = (await response.json()) as ChatApiResponse;
    } catch {
        throw new Error('Сервер вернул ответ в неподдерживаемом формате.');
    }

    if (!response.ok) {
        throw new Error(data.message || 'Не удалось отправить сообщение. Попробуйте ещё раз.');
    }

    if (!data.reply?.content || data.chat_id === undefined) {
        throw new Error('Ответ сервера не соответствует контракту AI-чата.');
    }

    return {
        chatId: String(data.chat_id),
        content: data.reply.content,
    };
}

export function startChat(content: string): Promise<ChatReply> {
    return postMessage('/api/core/chats', content);
}

export function sendChatMessage(
    chatId: string,
    content: string,
): Promise<ChatReply> {
    return postMessage(
        `/api/core/chats/${encodeURIComponent(chatId)}/messages`,
        content,
    );
}
