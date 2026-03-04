import { useCallback, useEffect, useRef, useState } from 'react';

export type WebSocketStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

export interface WebSocketMessage {
    type: string;
    channel?: string;
    data?: any;
}

export interface UseWebSocketOptions {
    url?: string;
    reconnectInterval?: number;
    maxReconnectAttempts?: number;
    channels?: string[];
    onMessage?: (message: WebSocketMessage) => void;
}

interface WebSocketHook {
    status: WebSocketStatus;
    connect: () => void;
    disconnect: () => void;
    subscribe: (channel: string) => void;
    unsubscribe: (channel: string) => void;
    send: (message: WebSocketMessage) => void;
    subscribedChannels: Set<string>;
}

function getDefaultWebSocketUrl(): string {
    if (typeof window === 'undefined') return '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/api/v2/events`;
}

export function useWebSocket(options: UseWebSocketOptions = {}): WebSocketHook {
    const {
        url = process.env.NEXT_PUBLIC_WS_URL || getDefaultWebSocketUrl(),
        reconnectInterval = 3000,
        maxReconnectAttempts = 5,
        channels = [],
        onMessage,
    } = options;

    const [status, setStatus] = useState<WebSocketStatus>('disconnected');
    const [subscribedChannels, setSubscribedChannels] = useState<Set<string>>(new Set());

    const wsRef = useRef<WebSocket | null>(null);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const reconnectAttemptsRef = useRef(0);
    const manualCloseRef = useRef(false);

    const cleanup = useCallback(() => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
            reconnectTimeoutRef.current = null;
        }
    }, []);

    const connect = useCallback(() => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            return;
        }

        manualCloseRef.current = false;
        setStatus('connecting');

        try {
            const ws = new WebSocket(url);

            ws.onopen = () => {
                console.log('[WebSocket] Connected');
                setStatus('connected');
                reconnectAttemptsRef.current = 0;

                // Resubscribe to channels on reconnect
                subscribedChannels.forEach((channel) => {
                    ws.send(JSON.stringify({ type: 'subscribe', channel }));
                });
            };

            ws.onclose = (event) => {
                console.log('[WebSocket] Disconnected', event.code, event.reason);
                wsRef.current = null;

                if (!manualCloseRef.current && status !== 'error') {
                    setStatus('disconnected');

                    // Attempt to reconnect
                    if (reconnectAttemptsRef.current < maxReconnectAttempts) {
                        reconnectAttemptsRef.current += 1;
                        console.log(`[WebSocket] Reconnecting (attempt ${reconnectAttemptsRef.current}/${maxReconnectAttempts})...`);

                        reconnectTimeoutRef.current = setTimeout(() => {
                            connect();
                        }, reconnectInterval);
                    } else {
                        console.log('[WebSocket] Max reconnect attempts reached');
                    }
                }
            };

            ws.onerror = (error) => {
                console.error('[WebSocket] Error', error);
                setStatus('error');
            };

            ws.onmessage = (event) => {
                try {
                    const message: WebSocketMessage = JSON.parse(event.data);
                    console.log('[WebSocket] Message received', message);

                    if (onMessage) {
                        onMessage(message);
                    }
                } catch (err) {
                    console.error('[WebSocket] Failed to parse message', err);
                }
            };

            wsRef.current = ws;
        } catch (error) {
            console.error('[WebSocket] Failed to create connection', error);
            setStatus('error');
        }
    }, [url, reconnectInterval, maxReconnectAttempts, onMessage]);

    const disconnect = useCallback(() => {
        manualCloseRef.current = true;
        cleanup();

        if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
        }

        setStatus('disconnected');
    }, [cleanup]);

    const subscribe = useCallback((channel: string) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'subscribe', channel }));
        }
        setSubscribedChannels((prev) => new Set(prev).add(channel));
    }, []);

    const unsubscribe = useCallback((channel: string) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'unsubscribe', channel }));
        }
        setSubscribedChannels((prev) => {
            const next = new Set(prev);
            next.delete(channel);
            return next;
        });
    }, []);

    const send = useCallback((message: WebSocketMessage) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify(message));
        } else {
            console.warn('[WebSocket] Cannot send message - not connected');
        }
    }, []);

    // Initial connection and channel subscriptions
    useEffect(() => {
        connect();

        return () => {
            disconnect();
        };
    }, [connect, disconnect]);

    // Subscribe to initial channels when connected
    useEffect(() => {
        if (status === 'connected' && channels.length > 0) {
            channels.forEach((channel) => {
                if (!subscribedChannels.has(channel)) {
                    subscribe(channel);
                }
            });
        }
    }, [status, channels, subscribe, subscribedChannels]);

    return {
        status,
        connect,
        disconnect,
        subscribe,
        unsubscribe,
        send,
        subscribedChannels,
    };
}

export default useWebSocket;
