'use client';

import { useState, useEffect } from 'react';
import { Clock } from 'lucide-react';

interface ContestTimerProps {
    startTime: string | Date;
    endTime: string | Date;
    onStatusChange?: (status: 'before' | 'ongoing' | 'ended') => void;
}

interface TimeLeft {
    days: number;
    hours: number;
    minutes: number;
    seconds: number;
    isExpired: boolean;
}

export function ContestTimer({ startTime, endTime, onStatusChange }: ContestTimerProps) {
    const [timeLeft, setTimeLeft] = useState<TimeLeft>({
        days: 0,
        hours: 0,
        minutes: 0,
        seconds: 0,
        isExpired: false
    });
    const [status, setStatus] = useState<'before' | 'ongoing' | 'ended'>('before');

    useEffect(() => {
        const start = new Date(startTime).getTime();
        const end = new Date(endTime).getTime();

        const calculateTimeLeft = (): TimeLeft => {
            const now = Date.now();
            let targetTime = start;
            let newStatus: 'before' | 'ongoing' | 'ended' = 'before';

            if (now >= start && now <= end) {
                targetTime = end;
                newStatus = 'ongoing';
            } else if (now > end) {
                return { days: 0, hours: 0, minutes: 0, seconds: 0, isExpired: true };
            }

            const difference = targetTime - now;
            newStatus = now < start ? 'before' : 'ongoing';

            return {
                days: Math.floor(difference / (1000 * 60 * 60 * 24)),
                hours: Math.floor((difference / (1000 * 60 * 60)) % 24),
                minutes: Math.floor((difference / 1000 / 60) % 60),
                seconds: Math.floor((difference / 1000) % 60),
                isExpired: false
            };
        };

        const updateTimer = () => {
            const newTimeLeft = calculateTimeLeft();
            setTimeLeft(newTimeLeft);

            // Determine status
            const now = Date.now();
            const start = new Date(startTime).getTime();
            const end = new Date(endTime).getTime();
            let newStatus: 'before' | 'ongoing' | 'ended' = 'before';

            if (now < start) {
                newStatus = 'before';
            } else if (now >= start && now <= end) {
                newStatus = 'ongoing';
            } else {
                newStatus = 'ended';
            }

            setStatus(newStatus);
            onStatusChange?.(newStatus);
        };

        updateTimer();
        const timer = setInterval(updateTimer, 1000);

        return () => clearInterval(timer);
    }, [startTime, endTime, onStatusChange]);

    const getStatusColor = () => {
        switch (status) {
            case 'before':
                return 'text-warning';
            case 'ongoing':
                return 'text-success';
            case 'ended':
                return 'text-muted-foreground';
        }
    };

    const getStatusText = () => {
        switch (status) {
            case 'before':
                return 'Starts in:';
            case 'ongoing':
                return 'Ends in:';
            case 'ended':
                return 'Ended';
        }
    };

    if (status === 'ended' || timeLeft.isExpired) {
        return (
            <div className={`flex items-center gap-2 ${getStatusColor()}`}>
                <Clock size={20} />
                <span className="font-bold">Contest Ended</span>
            </div>
        );
    }

    return (
        <div className={`flex items-center gap-3 ${getStatusColor()}`}>
            <Clock size={20} />
            <div className="flex flex-col">
                <span className="text-xs text-muted-foreground">{getStatusText()}</span>
                <div className="flex items-center gap-1 font-mono font-bold">
                    {timeLeft.days > 0 && (
                        <>
                            <span className="px-2 py-1 rounded bg-muted">{timeLeft.days}d</span>
                            <span>·</span>
                        </>
                    )}
                    <span className="px-2 py-1 rounded bg-muted">
                        {String(timeLeft.hours).padStart(2, '0')}h
                    </span>
                    <span>·</span>
                    <span className="px-2 py-1 rounded bg-muted">
                        {String(timeLeft.minutes).padStart(2, '0')}m
                    </span>
                    <span>·</span>
                    <span className="px-2 py-1 rounded bg-muted">
                        {String(timeLeft.seconds).padStart(2, '0')}s
                    </span>
                </div>
            </div>
        </div>
    );
}

export default ContestTimer;
