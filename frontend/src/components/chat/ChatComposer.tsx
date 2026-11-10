import { useState, type FormEvent, type KeyboardEvent } from 'react';

interface ChatComposerProps {
    disabled: boolean;
    onSend: (content: string) => Promise<boolean>;
}

const suggestions = [
    'Как увеличить маржинальность?',
    'Помоги составить план продаж',
    'Как оценить новую бизнес-идею?',
];

function ChatComposer({ disabled, onSend }: ChatComposerProps) {
    const [value, setValue] = useState('');

    const submit = async (event?: FormEvent<HTMLFormElement>) => {
        event?.preventDefault();
        const content = value.trim();
        if (!content || disabled) return;

        if (await onSend(content)) {
            setValue('');
        }
    };

    const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            void submit();
        }
    };

    return (
        <div className="composer-wrap">
            <div className="suggestion-list" aria-label="Примеры вопросов">
                {suggestions.map((suggestion) => (
                    <button
                        key={suggestion}
                        type="button"
                        disabled={disabled}
                        onClick={() => setValue(suggestion)}
                    >
                        {suggestion}
                    </button>
                ))}
            </div>

            <form className="chat-composer" onSubmit={submit}>
                <label className="visually-hidden" htmlFor="chat-input">
                    Ваш вопрос бизнес-ассистенту
                </label>
                <textarea
                    id="chat-input"
                    placeholder="Опишите задачу или задайте вопрос..."
                    rows={1}
                    value={value}
                    onChange={(event) => setValue(event.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={disabled}
                />
                <div className="composer-footer">
                    <span>Enter — отправить · Shift + Enter — новая строка</span>
                    <button
                        className="send-button"
                        type="submit"
                        disabled={disabled || !value.trim()}
                        aria-label="Отправить сообщение"
                    >
                        <span aria-hidden="true">↑</span>
                    </button>
                </div>
            </form>
            <p className="chat-disclaimer">
                Ответы ИИ носят информационный характер. Проверяйте важные решения и расчёты.
            </p>
        </div>
    );
}

export default ChatComposer;
