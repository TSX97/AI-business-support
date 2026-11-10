import { useState } from 'react';
import ChatComposer from './ChatComposer';
import ChatMessageList from './ChatMessageList';
import ChatSidebar from './ChatSidebar';
import { sendChatMessage, startChat } from '../../services/chat';
import type { ChatMessage } from '../../types/chat';
import './ChatPage.css';

const welcomeMessage: ChatMessage = {
    id: 'welcome',
    role: 'assistant',
    content:
        'Здравствуйте. Я помогу разобраться в бизнес-вопросе: от продаж и финансов до процессов и планирования. Опишите ситуацию, добавьте цифры и контекст — так рекомендации будут точнее.',
};

function ChatPage() {
    const [chatId, setChatId] = useState<string | null>(null);
    const [messages, setMessages] = useState<ChatMessage[]>([welcomeMessage]);
    const [waiting, setWaiting] = useState(false);
    const [error, setError] = useState('');

    const startNewChat = () => {
        if (waiting) return;
        setChatId(null);
        setMessages([welcomeMessage]);
        setError('');
    };

    const sendMessage = async (content: string): Promise<boolean> => {
        setError('');
        setWaiting(true);

        try {
            const reply = chatId
                ? await sendChatMessage(chatId, content)
                : await startChat(content);

            setChatId(reply.chatId);
            setMessages((current) => [
                ...current,
                {
                    id: `user-${Date.now()}`,
                    role: 'user',
                    content,
                },
                {
                    id: `assistant-${Date.now()}`,
                    role: 'assistant',
                    content: reply.content,
                },
            ]);
            return true;
        } catch (sendError) {
            setError(
                sendError instanceof Error
                    ? sendError.message
                    : 'Не удалось отправить сообщение.',
            );
            return false;
        } finally {
            setWaiting(false);
        }
    };

    return (
        <main className="chat-app">
            <ChatSidebar onNewChat={startNewChat} />

            <section className="chat-main">
                <header className="chat-topbar">
                    <div>
                        <span className="chat-topbar-eyebrow">ВАШ РАБОЧИЙ ИНСТРУМЕНТ</span>
                        <h1>Бизнес-ассистент</h1>
                    </div>
                    <a className="chat-account-link" href="/">
                        <span className="account-avatar">В</span>
                        <span>Аккаунт</span>
                    </a>
                </header>

                <section className="chat-workspace" aria-label="Диалог с AI-ассистентом">
                    <div className="conversation">
                        {messages.length === 1 && (
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

                        <ChatMessageList messages={messages} waiting={waiting} />
                    </div>

                    <div className="composer-area">
                        {error && (
                            <p className="chat-error" role="alert">
                                {error}
                            </p>
                        )}
                        <ChatComposer
                            key={chatId ?? 'new-chat'}
                            disabled={waiting}
                            onSend={sendMessage}
                        />
                    </div>
                </section>
            </section>
        </main>
    );
}

export default ChatPage;
