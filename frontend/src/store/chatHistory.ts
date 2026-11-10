import type { ChatMessage } from '../types/chat';

export interface ChatSession {
    id: string;
    backendId: string | null;
    title: string;
    messages: ChatMessage[];
    updatedAt: number;
}

export interface ChatHistoryState {
    chats: ChatSession[];
    activeChatId: string | null;
}

const storageKey = 'ai-business-support.chat-history.v1';

function isChatMessage(value: unknown): value is ChatMessage {
    if (typeof value !== 'object' || value === null) return false;

    const message = value as Record<string, unknown>;
    return (
        typeof message.id === 'string' &&
        (message.role === 'user' || message.role === 'assistant') &&
        typeof message.content === 'string'
    );
}

function isChatSession(value: unknown): value is ChatSession {
    if (typeof value !== 'object' || value === null) return false;

    const chat = value as Record<string, unknown>;
    return (
        typeof chat.id === 'string' &&
        (typeof chat.backendId === 'string' || chat.backendId === null) &&
        typeof chat.title === 'string' &&
        Array.isArray(chat.messages) &&
        chat.messages.every(isChatMessage) &&
        typeof chat.updatedAt === 'number'
    );
}

export function loadChatHistory(): ChatHistoryState {
    try {
        const saved = localStorage.getItem(storageKey);
        if (!saved) return { chats: [], activeChatId: null };

        const parsed: unknown = JSON.parse(saved);
        if (typeof parsed !== 'object' || parsed === null) {
            throw new Error('Stored chat history must be an object.');
        }

        const data = parsed as Record<string, unknown>;
        if (!Array.isArray(data.chats) || !data.chats.every(isChatSession)) {
            throw new Error('Stored chat history has an invalid shape.');
        }

        const activeChatId =
            typeof data.activeChatId === 'string' &&
            data.chats.some((chat) => chat.id === data.activeChatId)
                ? data.activeChatId
                : null;

        return { chats: data.chats, activeChatId };
    } catch (error) {
        console.warn('Unable to load saved chat history; starting with an empty history.', error);
        return { chats: [], activeChatId: null };
    }
}

export function saveChatHistory(state: ChatHistoryState): void {
    try {
        localStorage.setItem(storageKey, JSON.stringify(state));
    } catch (error) {
        console.warn('Unable to save chat history in this browser.', error);
    }
}

export function createLocalId(): string {
    return globalThis.crypto?.randomUUID?.() ??
        `chat-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
