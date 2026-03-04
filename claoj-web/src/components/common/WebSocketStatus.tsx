'use client';

import { useWebSocketContext } from '@/contexts/WebSocketContext';
import { Wifi, WifiOff, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface WebSocketStatusIndicatorProps {
    className?: string;
}

export function WebSocketStatusIndicator({ className }: WebSocketStatusIndicatorProps) {
    const { status } = useWebSocketContext();

    const getStatusInfo = () => {
        switch (status) {
            case 'connected':
                return {
                    icon: <Wifi size={16} />,
                    label: 'Connected',
                    color: 'text-emerald-500',
                    bg: 'bg-emerald-500/10',
                    border: 'border-emerald-500/20',
                };
            case 'connecting':
                return {
                    icon: <Loader2 size={16} className="animate-spin" />,
                    label: 'Connecting...',
                    color: 'text-amber-500',
                    bg: 'bg-amber-500/10',
                    border: 'border-amber-500/20',
                };
            case 'error':
                return {
                    icon: <WifiOff size={16} />,
                    label: 'Connection Error',
                    color: 'text-red-500',
                    bg: 'bg-red-500/10',
                    border: 'border-red-500/20',
                };
            case 'disconnected':
            default:
                return {
                    icon: <WifiOff size={16} />,
                    label: 'Disconnected',
                    color: 'text-zinc-500',
                    bg: 'bg-zinc-500/10',
                    border: 'border-zinc-500/20',
                };
        }
    };

    const statusInfo = getStatusInfo();

    return (
        <div
            className={cn(
                'flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium transition-colors',
                statusInfo.bg,
                statusInfo.color,
                className
            )}
            title={statusInfo.label}
        >
            {statusInfo.icon}
            <span className="hidden sm:inline">{statusInfo.label}</span>
        </div>
    );
}

export default WebSocketStatusIndicator;
