'use client';

import React, { createContext, useContext, useCallback, useMemo } from 'react';
import { useWebSocket, WebSocketStatus, WebSocketMessage } from '../hooks/useWebSocket';

interface WebSocketContextValue {
    status: WebSocketStatus;
    connect: () => void;
    disconnect: () => void;
    subscribe: (channel: string) => void;
    unsubscribe: (channel: string) => void;
    send: (message: WebSocketMessage) => void;
    subscribedChannels: Set<string>;
}

const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

interface WebSocketProviderProps {
    children: React.ReactNode;
    url?: string;
    reconnectInterval?: number;
    maxReconnectAttempts?: number;
    defaultChannels?: string[];
    onMessage?: (message: WebSocketMessage) => void;
}

export function WebSocketProvider({
    children,
    url,
    reconnectInterval,
    maxReconnectAttempts,
    defaultChannels = [],
    onMessage,
}: WebSocketProviderProps) {
    const {
        status,
        connect,
        disconnect,
        subscribe,
        unsubscribe,
        send,
        subscribedChannels,
    } = useWebSocket({
        url,
        reconnectInterval,
        maxReconnectAttempts,
        channels: defaultChannels,
        onMessage,
    });

    const value = useMemo(() => ({
        status,
        connect,
        disconnect,
        subscribe,
        unsubscribe,
        send,
        subscribedChannels,
    }), [status, connect, disconnect, subscribe, unsubscribe, send, subscribedChannels]);

    return (
        <WebSocketContext.Provider value={value}>
            {children}
        </WebSocketContext.Provider>
    );
}

export function useWebSocketContext() {
    const context = useContext(WebSocketContext);
    if (context === undefined) {
        throw new Error('useWebSocketContext must be used within a WebSocketProvider');
    }
    return context;
}

export { WebSocketContext };
export default WebSocketProvider;
