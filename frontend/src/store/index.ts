import { configureStore } from '@reduxjs/toolkit';
import chatsReducer from './chatsSlice';
import { saveChatHistory } from './chatHistory';

export const store = configureStore({
    reducer: {
        chatHistory: chatsReducer,
    },
});

store.subscribe(() => {
    saveChatHistory(store.getState().chatHistory);
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
