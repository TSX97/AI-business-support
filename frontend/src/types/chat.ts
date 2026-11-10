export type ChatRole = 'user' | 'assistant';

export interface ChatMessage {
    id: string;
    role: ChatRole;
    content: string;
}

export interface ChatReply {
    chatId: string;
    content: string;
}
