'use client';

import React, { createContext, useContext, useEffect, useMemo, useRef } from 'react';
import { useWebSocket, WebSocketStatus, WebSocketMessage } from '../hooks/useWebSocket';

interface WebSocketContextValue {
    status: WebSocketStatus;
    connect: () => void;
    disconnect: () => void;
    subscribe: (channel: string) => void;
    unsubscribe: (channel: string) => void;
    send: (message: WebSocketMessage) => void;
    subscribedChannels: Set<string>;
    addListener: (fn: (message: WebSocketMessage) => void) => () => void;
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
        addListener,
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
        addListener,
    }), [status, connect, disconnect, subscribe, unsubscribe, send, subscribedChannels, addListener]);

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

/**
 * Subscribes to one backend event channel while mounted and calls onEvent with
 * each message's data. A falsy channel subscribes to nothing.
 */
export function useEventChannel(channel: string | undefined | null, onEvent: (data: unknown) => void) {
    const { subscribe, unsubscribe, addListener } = useWebSocketContext();
    const handler = useRef(onEvent);
    useEffect(() => { handler.current = onEvent; });
    useEffect(() => {
        if (!channel) return;
        subscribe(channel);
        const off = addListener((m) => { if (m.channel === channel) handler.current(m.data); });
        return () => { off(); unsubscribe(channel); };
    }, [channel, subscribe, unsubscribe, addListener]);
}

export { WebSocketContext };
export default WebSocketProvider;
