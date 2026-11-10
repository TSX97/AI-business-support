import { createSlice, type PayloadAction } from '@reduxjs/toolkit';
import type { ChatMessage } from '../types/chat';
import {
    createLocalId,
    loadChatHistory,
    type ChatHistoryState,
    type ChatSession,
} from './chatHistory';

interface ChatExchangePayload {
    chatId: string;
    backendId: string;
    userContent: string;
    assistantContent: string;
    updatedAt: number;
}

function makeChatTitle(content: string): string {
    const title = content.replace(/\s+/g, ' ').trim();
    return title.length > 38 ? `${title.slice(0, 37).trimEnd()}…` : title;
}

const initialState: ChatHistoryState = loadChatHistory();

const chatsSlice = createSlice({
    name: 'chatHistory',
    initialState,
    reducers: {
        chatCreated: {
            reducer(state, action: PayloadAction<ChatSession>) {
                state.chats.unshift(action.payload);
                state.activeChatId = action.payload.id;
            },
            prepare() {
                const now = Date.now();
                return {
                    payload: {
                        id: createLocalId(),
                        backendId: null,
                        title: 'Новый диалог',
                        messages: [],
                        updatedAt: now,
                    },
                };
            },
        },
        chatSelected(state, action: PayloadAction<string>) {
            if (state.chats.some((chat) => chat.id === action.payload)) {
                state.activeChatId = action.payload;
            }
        },
        chatDeleted(state, action: PayloadAction<string>) {
            state.chats = state.chats.filter((chat) => chat.id !== action.payload);
            if (state.activeChatId === action.payload) {
                state.activeChatId = state.chats[0]?.id ?? null;
            }
        },
        exchangeReceived(state, action: PayloadAction<ChatExchangePayload>) {
            const chat = state.chats.find(
                (item) => item.id === action.payload.chatId,
            );
            if (!chat) return;

            if (chat.messages.length === 0) {
                chat.title = makeChatTitle(action.payload.userContent);
            }

            const timestamp = action.payload.updatedAt;
            const userMessage: ChatMessage = {
                id: createLocalId(),
                role: 'user',
                content: action.payload.userContent,
            };
            const assistantMessage: ChatMessage = {
                id: createLocalId(),
                role: 'assistant',
                content: action.payload.assistantContent,
            };

            chat.backendId = action.payload.backendId;
            chat.messages.push(userMessage, assistantMessage);
            chat.updatedAt = timestamp;
            state.chats.sort((left, right) => right.updatedAt - left.updatedAt);
        },
    },
});

export const {
    chatCreated,
    chatDeleted,
    chatSelected,
    exchangeReceived,
} = chatsSlice.actions;

export default chatsSlice.reducer;
