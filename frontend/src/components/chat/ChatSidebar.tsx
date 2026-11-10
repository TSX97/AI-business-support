interface ChatSidebarProps {
    onNewChat: () => void;
}

function ChatSidebar({ onNewChat }: ChatSidebarProps) {
    return (
        <aside className="chat-sidebar">
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
                <span className="sidebar-section-label">РАБОЧЕЕ ПРОСТРАНСТВО</span>
                <p className="history-empty">
                    История диалогов появится здесь после подключения сервера.
                </p>
            </div>

            <div className="sidebar-bottom">
                <div className="sidebar-help-mark" aria-hidden="true">?</div>
                <div>
                    <strong>Помощь по бизнесу</strong>
                    <span>Практично. По делу.</span>
                </div>
            </div>
        </aside>
    );
}

export default ChatSidebar;
