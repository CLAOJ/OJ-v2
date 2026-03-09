'use client';

import { useState, useEffect } from 'react';

// Time conversion constants
const MS_PER_SECOND = 1000;
const MS_PER_MINUTE = MS_PER_SECOND * 60;
const MS_PER_HOUR = MS_PER_MINUTE * 60;

export function useContestTimer(inContest: boolean, contestEndTime: Date | null) {
    const [currentTime, setCurrentTime] = useState<Date | null>(null);

    // Timer countdown for contest
    useEffect(() => {
        if (!inContest || !contestEndTime) return;

        const timer = setInterval(() => {
            const now = new Date();
            setCurrentTime(now);

            if (now >= contestEndTime) {
                // Contest ended
            }
        }, 1000);

        return () => clearInterval(timer);
    }, [inContest, contestEndTime]);

    const formatTimeRemaining = () => {
        if (!currentTime || !contestEndTime) return '';

        const diff = contestEndTime.getTime() - currentTime.getTime();
        if (diff <= 0) return 'Ended';

        const hours = Math.floor(diff / MS_PER_HOUR);
        const minutes = Math.floor((diff % MS_PER_HOUR) / MS_PER_MINUTE);
        const seconds = Math.floor((diff % MS_PER_MINUTE) / MS_PER_SECOND);

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    };

    return {
        currentTime,
        formatTimeRemaining,
    };
}
