import { useEffect, useRef } from 'react';
import type { ChatMessage } from '../../types/chat';

interface ChatMessageListProps {
    messages: ChatMessage[];
    waiting: boolean;
}

function ChatMessageList({ messages, waiting }: ChatMessageListProps) {
    const bottomRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
    }, [messages, waiting]);

    return (
        <div className="chat-messages" aria-live="polite">
            {messages.map((message) => (
                <article
                    className={`chat-message ${message.role}`}
                    key={message.id}
                >
                    {message.role === 'assistant' && (
                        <div className="assistant-avatar" aria-hidden="true">AI</div>
                    )}
                    <div className="message-content">
                        {message.role === 'assistant' && (
                            <span className="message-author">AI-ассистент</span>
                        )}
                        <p>{message.content}</p>
                    </div>
                    {message.role === 'user' && (
                        <div className="user-avatar" aria-hidden="true">ВЫ</div>
                    )}
                </article>
            ))}

            {waiting && (
                <div className="chat-message assistant" role="status">
                    <div className="assistant-avatar" aria-hidden="true">AI</div>
                    <div className="typing-indicator" aria-label="Ассистент готовит ответ">
                        <span />
                        <span />
                        <span />
                    </div>
                </div>
            )}
            <div ref={bottomRef} />
        </div>
    );
}

export default ChatMessageList;
