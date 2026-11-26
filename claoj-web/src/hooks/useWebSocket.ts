import { useCallback, useEffect, useRef, useState } from 'react';

export type WebSocketStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

export interface WebSocketMessage {
    type: string;
    channel?: string;
    data?: unknown;
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
    /** Calls fn for every incoming message; returns the remover. */
    addListener: (fn: (message: WebSocketMessage) => void) => () => void;
}

function getDefaultWebSocketUrl(): string {
    if (typeof window === 'undefined') return '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/api/events`;
}

export function useWebSocket(options: UseWebSocketOptions = {}): WebSocketHook {
    const {
        url = process.env.NEXT_PUBLIC_WS_URL || getDefaultWebSocketUrl(),
        reconnectInterval = 3000,
        maxReconnectAttempts = Infinity, // a deploy restart must not kill live updates for good
        channels = [],
        onMessage,
    } = options;

    const [status, setStatus] = useState<WebSocketStatus>('connecting'); // connect() runs on mount
    const [subscribedChannels, setSubscribedChannels] = useState<Set<string>>(new Set());

    const wsRef = useRef<WebSocket | null>(null);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const reconnectAttemptsRef = useRef(0);
    const manualCloseRef = useRef(false);
    const listenersRef = useRef(new Set<(message: WebSocketMessage) => void>());
    // The reconnect timer calls the latest connect, so connect never references itself.
    const connectRef = useRef<() => void>(() => {});
    // Read by onopen, which outlives the render that created it.
    const subscribedRef = useRef(subscribedChannels);
    useEffect(() => { subscribedRef.current = subscribedChannels; }, [subscribedChannels]);

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

        try {
            const ws = new WebSocket(url);

            ws.onopen = () => {
                setStatus('connected');
                reconnectAttemptsRef.current = 0;

                // Resubscribe to channels on reconnect
                subscribedRef.current.forEach((channel) => {
                    ws.send(JSON.stringify({ type: 'subscribe', channel }));
                });
            };

            ws.onclose = (event) => {
                wsRef.current = null;

                if (!manualCloseRef.current) {
                    setStatus('disconnected');

                    // Attempt to reconnect
                    if (reconnectAttemptsRef.current < maxReconnectAttempts) {
                        reconnectAttemptsRef.current += 1;

                        reconnectTimeoutRef.current = setTimeout(() => {
                            setStatus('connecting');
                            connectRef.current();
                        }, Math.min(reconnectInterval * reconnectAttemptsRef.current, 30000));
                    }
                }
            };

            ws.onerror = (error) => {
                setStatus('error');
            };

            ws.onmessage = (event) => {
                try {
                    const message: WebSocketMessage = JSON.parse(event.data);

                    onMessage?.(message);
                    listenersRef.current.forEach((fn) => fn(message));
                } catch (err) {
                    // Failed to parse message - silently ignore
                }
            };

            wsRef.current = ws;
        } catch {
            // Only an invalid URL throws here. Report it asynchronously, like onerror,
            // so connect() never sets state synchronously inside the mount effect.
            queueMicrotask(() => setStatus('error'));
        }
    }, [url, reconnectInterval, maxReconnectAttempts, onMessage]);
    useEffect(() => { connectRef.current = connect; }, [connect]);

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
        }
        // Silently ignore if not connected - will be handled by connection state
    }, []);

    const addListener = useCallback((fn: (message: WebSocketMessage) => void) => {
        listenersRef.current.add(fn);
        return () => { listenersRef.current.delete(fn); };
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
        addListener,
    };
}

export default useWebSocket;
