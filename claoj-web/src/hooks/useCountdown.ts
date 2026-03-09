'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';

interface TimeRemaining {
    days: number;
    hours: number;
    minutes: number;
    seconds: number;
    totalMilliseconds: number;
    isExpired: boolean;
}

interface UseCountdownOptions {
    targetDate?: Date | string | number;
    duration?: number; // in milliseconds
    interval?: number; // in milliseconds, default 1000
    onExpire?: () => void;
    autoStart?: boolean;
}

interface UseCountdownReturn extends TimeRemaining {
    start: () => void;
    pause: () => void;
    reset: () => void;
    isRunning: boolean;
    formatted: string;
}

// Time conversion constants
const MS_PER_SECOND = 1000;
const MS_PER_MINUTE = MS_PER_SECOND * 60;
const MS_PER_HOUR = MS_PER_MINUTE * 60;
const MS_PER_DAY = MS_PER_HOUR * 24;

function calculateTimeRemaining(targetTime: number): TimeRemaining {
    const now = Date.now();
    const diff = targetTime - now;

    if (diff <= 0) {
        return {
            days: 0,
            hours: 0,
            minutes: 0,
            seconds: 0,
            totalMilliseconds: 0,
            isExpired: true,
        };
    }

    return {
        days: Math.floor(diff / MS_PER_DAY),
        hours: Math.floor((diff % MS_PER_DAY) / MS_PER_HOUR),
        minutes: Math.floor((diff % MS_PER_HOUR) / MS_PER_MINUTE),
        seconds: Math.floor((diff % MS_PER_MINUTE) / MS_PER_SECOND),
        totalMilliseconds: diff,
        isExpired: false,
    };
}

/**
 * Custom hook for countdown timer functionality
 *
 * @example
 * // For a contest ending at a specific time
 * const countdown = useCountdown({ targetDate: contestEndTime, onExpire: () => alert('Contest ended!') });
 *
 * // Display: {countdown.formatted}
 *
 * @example
 * // For a duration-based countdown
 * const countdown = useCountdown({ duration: 5 * 60 * 1000 }); // 5 minutes
 * countdown.start(); // Start the countdown
 */
export function useCountdown(options: UseCountdownOptions = {}): UseCountdownReturn {
    const {
        targetDate,
        duration,
        interval = 1000,
        onExpire,
        autoStart = true,
    } = options;

    const getInitialTargetTime = useCallback(() => {
        if (targetDate) {
            return new Date(targetDate).getTime();
        }
        if (duration) {
            return Date.now() + duration;
        }
        return Date.now();
    }, [targetDate, duration]);

    const [targetTime, setTargetTime] = useState(getInitialTargetTime);
    const [isRunning, setIsRunning] = useState(autoStart);
    const [timeRemaining, setTimeRemaining] = useState<TimeRemaining>(() =>
        calculateTimeRemaining(targetTime)
    );

    useEffect(() => {
        if (!isRunning) return;

        const timer = setInterval(() => {
            const remaining = calculateTimeRemaining(targetTime);
            setTimeRemaining(remaining);

            if (remaining.isExpired) {
                setIsRunning(false);
                onExpire?.();
            }
        }, interval);

        return () => clearInterval(timer);
    }, [isRunning, targetTime, interval, onExpire]);

    const start = useCallback(() => {
        if (timeRemaining.isExpired && duration) {
            // Reset if expired and using duration mode
            setTargetTime(Date.now() + duration);
        }
        setIsRunning(true);
    }, [timeRemaining.isExpired, duration]);

    const pause = useCallback(() => {
        setIsRunning(false);
    }, []);

    const reset = useCallback(() => {
        setIsRunning(false);
        const newTargetTime = getInitialTargetTime();
        setTargetTime(newTargetTime);
        setTimeRemaining(calculateTimeRemaining(newTargetTime));
    }, [getInitialTargetTime]);

    const formatted = useMemo(() => {
        const { days, hours, minutes, seconds } = timeRemaining;

        if (days > 0) {
            return `${days}d ${hours.toString().padStart(2, '0')}h ${minutes.toString().padStart(2, '0')}m`;
        }

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }, [timeRemaining]);

    return {
        ...timeRemaining,
        start,
        pause,
        reset,
        isRunning,
        formatted,
    };
}

/**
 * Hook specifically for contest countdowns
 * Formats time as HH:MM:SS for contests
 */
export function useContestCountdown(
    endTime: Date | string | number | null | undefined,
    options: { onExpire?: () => void } = {}
): { formatted: string; isExpired: boolean; timeRemaining: TimeRemaining | null } {
    const [isClient, setIsClient] = useState(false);

    useEffect(() => {
        setIsClient(true);
    }, []);

    const countdown = useCountdown({
        targetDate: endTime || undefined,
        onExpire: options.onExpire,
        autoStart: !!endTime,
    });

    // Server-safe formatted time
    const formatted = useMemo(() => {
        if (!isClient || !endTime) return '--:--:--';
        return countdown.formatted;
    }, [isClient, endTime, countdown.formatted]);

    return {
        formatted,
        isExpired: countdown.isExpired,
        timeRemaining: isClient ? countdown : null,
    };
}
