import { useState } from 'react';
import { sendChatMessage, startChat } from '../../services/chat';
import type { ChatMessage } from '../../types/chat';
import { useAppDispatch, useAppSelector } from '../../store/hooks';
import {
    chatCreated,
    chatDeleted,
    chatSelected,
    exchangeReceived,
} from '../../store/chatsSlice';
import ChatComposer from './ChatComposer';
import ChatMessageList from './ChatMessageList';
import ChatSidebar from './ChatSidebar';
import './ChatPage.css';

const welcomeMessage: ChatMessage = {
    id: 'welcome',
    role: 'assistant',
    content:
        'Здравствуйте. Я помогу разобраться в бизнес-вопросе: от продаж и финансов до процессов и планирования. Опишите ситуацию, добавьте цифры и контекст — так рекомендации будут точнее.',
};

function ChatPage() {
    const dispatch = useAppDispatch();
    const { chats, activeChatId } = useAppSelector(
        (state) => state.chatHistory,
    );
    const activeChat = chats.find((chat) => chat.id === activeChatId);
    const [pendingChatIds, setPendingChatIds] = useState<string[]>([]);
    const [errorsByChatId, setErrorsByChatId] = useState<Record<string, string>>({});
    const [sidebarOpen, setSidebarOpen] = useState(false);

    const startNewChat = () => {
        dispatch(chatCreated());
        setSidebarOpen(false);
    };

    const selectChat = (chatId: string) => {
        dispatch(chatSelected(chatId));
        setSidebarOpen(false);
    };

    const deleteChat = (chatId: string) => {
        const chat = chats.find((item) => item.id === chatId);
        if (!chat) return;

        const shouldDelete = window.confirm(
            `Удалить диалог «${chat.title}»? История будет удалена из этого браузера.`,
        );
        if (!shouldDelete) return;

        dispatch(chatDeleted(chatId));
        setErrorsByChatId((current) => {
            const next = { ...current };
            delete next[chatId];
            return next;
        });
    };

    const sendMessage = async (content: string): Promise<boolean> => {
        let targetChat = activeChat;
        if (!targetChat) {
            targetChat = dispatch(chatCreated()).payload;
        }

        const targetChatId = targetChat.id;
        if (pendingChatIds.includes(targetChatId)) return false;

        setPendingChatIds((current) => [...current, targetChatId]);
        setErrorsByChatId((current) => {
            const next = { ...current };
            delete next[targetChatId];
            return next;
        });

        try {
            const reply = targetChat.backendId
                ? await sendChatMessage(targetChat.backendId, content)
                : await startChat(content);

            dispatch(
                exchangeReceived({
                    chatId: targetChatId,
                    backendId: reply.chatId,
                    userContent: content,
                    assistantContent: reply.content,
                    updatedAt: Date.now(),
                }),
            );
            return true;
        } catch (sendError) {
            setErrorsByChatId((current) => ({
                ...current,
                [targetChatId]:
                    sendError instanceof Error
                        ? sendError.message
                        : 'Не удалось отправить сообщение.',
            }));
            return false;
        } finally {
            setPendingChatIds((current) =>
                current.filter((chatId) => chatId !== targetChatId),
            );
        }
    };

    const isWaiting = activeChat
        ? pendingChatIds.includes(activeChat.id)
        : false;
    const messages = activeChat?.messages ?? [];

    return (
        <main className="chat-app">
            <ChatSidebar
                chats={chats}
                activeChatId={activeChatId}
                pendingChatIds={pendingChatIds}
                isOpen={sidebarOpen}
                onNewChat={startNewChat}
                onSelectChat={selectChat}
                onDeleteChat={deleteChat}
                onClose={() => setSidebarOpen(false)}
            />

            <section className="chat-main">
                <header className="chat-topbar">
                    <button
                        className="chat-menu-button"
                        type="button"
                        aria-label="Открыть список диалогов"
                        aria-expanded={sidebarOpen}
                        onClick={() => setSidebarOpen(true)}
                    >
                        <span aria-hidden="true">☰</span>
                    </button>
                    <div className="chat-topbar-title">
                        <span className="chat-topbar-eyebrow">ВАШ РАБОЧИЙ ИНСТРУМЕНТ</span>
                        <h1>{activeChat?.title ?? 'Бизнес-ассистент'}</h1>
                    </div>
                    <a className="chat-account-link" href="/">
                        <span className="account-avatar">В</span>
                        <span>Аккаунт</span>
                    </a>
                </header>

                <section className="chat-workspace" aria-label="Диалог с AI-ассистентом">
                    <div className="conversation">
                        {messages.length === 0 && (
                            <div className="conversation-intro">
                                <span className="conversation-kicker">
                                    AI ДЛЯ ВАШЕГО БИЗНЕСА
                                </span>
                                <h2>С чего начнём?</h2>
                                <p>
                                    Задайте конкретный вопрос — разберём вводные и наметим
                                    практические следующие шаги.
                                </p>
                            </div>
                        )}

                        <ChatMessageList
                            messages={messages.length === 0 ? [welcomeMessage] : messages}
                            waiting={isWaiting}
                        />
                    </div>

                    <div className="composer-area">
                        {activeChat && errorsByChatId[activeChat.id] && (
                            <p className="chat-error" role="alert">
                                {errorsByChatId[activeChat.id]}
                            </p>
                        )}
                        <ChatComposer
                            key={activeChatId ?? 'new-chat'}
                            disabled={isWaiting}
                            onSend={sendMessage}
                        />
                    </div>
                </section>
            </section>
        </main>
    );
}

export default ChatPage;
