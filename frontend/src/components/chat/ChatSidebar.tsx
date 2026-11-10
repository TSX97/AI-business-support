import type { ChatSession } from '../../store/chatHistory';

interface ChatSidebarProps {
    chats: ChatSession[];
    activeChatId: string | null;
    pendingChatIds: string[];
    isOpen: boolean;
    onNewChat: () => void;
    onSelectChat: (chatId: string) => void;
    onDeleteChat: (chatId: string) => void;
    onClose: () => void;
}

function formatChatDate(timestamp: number): string {
    return new Intl.DateTimeFormat('ru-RU', {
        hour: '2-digit',
        minute: '2-digit',
    }).format(timestamp);
}

function ChatSidebar({
    chats,
    activeChatId,
    pendingChatIds,
    isOpen,
    onNewChat,
    onSelectChat,
    onDeleteChat,
    onClose,
}: ChatSidebarProps) {
    return (
        <>
            {isOpen && (
                <button
                    className="sidebar-backdrop"
                    type="button"
                    aria-label="Закрыть список диалогов"
                    onClick={onClose}
                />
            )}
            <aside className={`chat-sidebar${isOpen ? ' is-open' : ''}`}>
                <a className="chat-brand" href="/" aria-label="На главную">
                    <span className="chat-brand-mark">AI</span>
                    <span>
                        <strong>Бизнес</strong>
                        <small>ПОМОЩНИК</small>
                    </span>
                </a>

                <button className="new-chat-button" onClick={onNewChat} type="button">
                    <span aria-hidden="true">＋</span>
                    Новый диалог
                </button>

                <div className="chat-history">
                    <div className="chat-history-heading">
                        <span className="sidebar-section-label">ДИАЛОГИ</span>
                        {chats.length > 0 && (
                            <span className="chat-count" aria-label={`${chats.length} диалогов`}>
                                {chats.length}
                            </span>
                        )}
                    </div>

                    {chats.length === 0 ? (
                        <p className="history-empty">
                            Здесь будут ваши диалоги. История пока хранится только в этом браузере.
                        </p>
                    ) : (
                        <ul className="chat-history-list" aria-label="История диалогов">
                            {chats.map((chat) => {
                                const isActive = chat.id === activeChatId;
                                const isPending = pendingChatIds.includes(chat.id);

                                return (
                                    <li className="chat-history-item" key={chat.id}>
                                        <button
                                            className={`chat-history-select${isActive ? ' is-active' : ''}`}
                                            type="button"
                                            aria-current={isActive ? 'page' : undefined}
                                            onClick={() => onSelectChat(chat.id)}
                                        >
                                            <span className="chat-history-title">{chat.title}</span>
                                            <span className="chat-history-date">
                                                {formatChatDate(chat.updatedAt)}
                                            </span>
                                        </button>
                                        <button
                                            className="chat-history-delete"
                                            type="button"
                                            aria-label={`Удалить диалог «${chat.title}»`}
                                            title="Удалить диалог"
                                            disabled={isPending}
                                            onClick={() => onDeleteChat(chat.id)}
                                        >
                                            <span aria-hidden="true">×</span>
                                        </button>
                                    </li>
                                );
                            })}
                        </ul>
                    )}
                </div>

                <div className="sidebar-bottom">
                    <div className="sidebar-help-mark" aria-hidden="true">?</div>
                    <div>
                        <strong>Помощь по бизнесу</strong>
                        <span>Практично. По делу.</span>
                    </div>
                </div>
            </aside>
        </>
    );
}

export default ChatSidebar;
