'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { contestCalendarApi } from '@/lib/api';
import { ContestCalendarItem, ContestCalendarResponse } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    Calendar,
    ChevronLeft,
    ChevronRight,
    Trophy,
    Star,
    Clock,
    Zap
} from 'lucide-react';
import dayjs from 'dayjs';
import isBetween from 'dayjs/plugin/isBetween';

dayjs.extend(isBetween);

const MONTH_NAMES = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
];

const WEEKDAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export default function ContestCalendarPage() {
    const t = useTranslations('Contest');
    const today = dayjs();
    const [currentYear, setCurrentYear] = useState(today.year());
    const [currentMonth, setCurrentMonth] = useState(today.month() + 1); // 1-indexed

    const { data: calendarData, isLoading } = useQuery({
        queryKey: ['contest-calendar', currentYear, currentMonth],
        queryFn: async () => {
            const res = await contestCalendarApi.getCalendar(currentYear, currentMonth);
            return res.data;
        }
    });

    const goToMonth = (year: number, month: number) => {
        setCurrentYear(year);
        setCurrentMonth(month);
    };

    const prevMonth = () => {
        if (currentMonth === 1) {
            goToMonth(currentYear - 1, 12);
        } else {
            goToMonth(currentYear, currentMonth - 1);
        }
    };

    const nextMonth = () => {
        if (currentMonth === 12) {
            goToMonth(currentYear + 1, 1);
        } else {
            goToMonth(currentYear, currentMonth + 1);
        }
    };

    const goToToday = () => {
        goToMonth(today.year(), today.month() + 1);
    };

    const getContestsForDay = (day: number, contests: ContestCalendarItem[] = []): ContestCalendarItem[] => {
        if (!contests.length && calendarData) {
            contests = calendarData.contests;
        }
        return contests.filter(c => c.day === day);
    };

    const getContestStatus = (contest: ContestCalendarItem) => {
        const now = dayjs();
        const start = dayjs(contest.start_time);
        const end = dayjs(contest.end_time);

        if (now.isBefore(start)) return 'upcoming';
        if (now.isAfter(end)) return 'past';
        return 'ongoing';
    };

    const renderCalendarGrid = () => {
        if (!calendarData) return null;

        const { days_in_month, first_day_of_week, contests } = calendarData;

        // Create array of days
        const days: (number | null)[] = [];

        // Add empty cells for days before the first day of the month
        for (let i = 0; i < first_day_of_week; i++) {
            days.push(null);
        }

        // Add days of the month
        for (let day = 1; day <= days_in_month; day++) {
            days.push(day);
        }

        // Calculate number of rows needed (6 rows max for any month)
        const totalCells = Math.ceil(days.length / 7) * 7;
        while (days.length < totalCells) {
            days.push(null);
        }

        return (
            <div className="grid grid-cols-7 gap-px bg-border rounded-xl overflow-hidden border">
                {/* Weekday headers */}
                {WEEKDAY_NAMES.map(day => (
                    <div
                        key={day}
                        className="bg-muted/50 py-3 text-center text-xs font-bold uppercase tracking-wider text-muted-foreground"
                    >
                        {day}
                    </div>
                ))}

                {/* Calendar days */}
                {days.map((day, index) => {
                    const dayContests = day ? getContestsForDay(day, contests) : [];
                    const isToday = day &&
                        currentYear === today.year() &&
                        currentMonth === today.month() + 1 &&
                        day === today.date();

                    return (
                        <div
                            key={index}
                            className={`min-h-[100px] bg-card p-2 transition-colors ${day ? 'hover:bg-muted/20' : ''}`}
                        >
                            {day && (
                                <div className="space-y-1">
                                    <div className="flex items-center justify-between mb-1">
                                        <span
                                            className={`text-sm font-bold w-7 h-7 flex items-center justify-center rounded-full ${isToday
                                                    ? 'bg-primary text-primary-foreground'
                                                    : 'text-foreground'
                                                }`}
                                        >
                                            {day}
                                        </span>
                                    </div>

                                    <div className="space-y-1">
                                        {dayContests.slice(0, 3).map(contest => {
                                            const status = getContestStatus(contest);
                                            const statusColors = {
                                                ongoing: 'bg-emerald-500/10 text-emerald-600 border-emerald-200',
                                                upcoming: 'bg-blue-500/10 text-blue-600 border-blue-200',
                                                past: 'bg-muted text-muted-foreground border-muted'
                                            };

                                            return (
                                                <Link
                                                    key={contest.key}
                                                    href={`/contests/${contest.key}`}
                                                    className={`block p-1.5 rounded-md border text-[10px] font-medium transition-all hover:shadow-md ${statusColors[status]}`}
                                                >
                                                    <div className="flex items-center gap-1">
                                                        {contest.is_rated && (
                                                            <Star size={8} className="fill-current shrink-0" />
                                                        )}
                                                        <span className="truncate">{contest.name}</span>
                                                    </div>
                                                    <div className="flex items-center gap-1 mt-0.5 text-[9px] opacity-70">
                                                        <Clock size={7} className="shrink-0" />
                                                        <span className="truncate">
                                                            {dayjs(contest.start_time).format('HH:mm')}
                                                        </span>
                                                    </div>
                                                </Link>
                                            );
                                        })}

                                        {dayContests.length > 3 && (
                                            <div className="text-[9px] text-muted-foreground text-center">
                                                +{dayContests.length - 3} more
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
        );
    };

    const renderContestList = () => {
        if (!calendarData || calendarData.contests.length === 0) {
            return (
                <div className="bg-muted/20 rounded-xl p-6 text-center text-muted-foreground border border-dashed">
                    <Calendar size={48} className="mx-auto mb-3 opacity-50" />
                    <p className="font-medium">No contests scheduled for {MONTH_NAMES[currentMonth - 1]} {currentYear}</p>
                </div>
            );
        }

        return (
            <div className="space-y-2">
                <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground mb-3">
                    Contests in {MONTH_NAMES[currentMonth - 1]} {currentYear}
                </h3>
                {calendarData.contests.map(contest => {
                    const status = getContestStatus(contest);
                    const statusConfig = {
                        ongoing: {
                            label: 'Ongoing',
                            color: 'bg-emerald-500 text-white',
                            icon: Zap
                        },
                        upcoming: {
                            label: 'Upcoming',
                            color: 'bg-blue-500 text-white',
                            icon: Clock
                        },
                        past: {
                            label: 'Past',
                            color: 'bg-muted text-muted-foreground',
                            icon: Calendar
                        }
                    };

                    const config = statusConfig[status];
                    const StatusIcon = config.icon;

                    return (
                        <Link
                            key={contest.key}
                            href={`/contests/${contest.key}`}
                            className="block p-4 bg-card rounded-xl border hover:shadow-md transition-all group"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-2 mb-1">
                                        <Trophy size={16} className="text-primary shrink-0" />
                                        <h4 className="font-bold text-base group-hover:text-primary transition-colors truncate">
                                            {contest.name}
                                        </h4>
                                    </div>
                                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                                        <span className="font-mono">
                                            {dayjs(contest.start_time).format('MMM D, YYYY HH:mm')}
                                        </span>
                                        <span>-</span>
                                        <span>
                                            {dayjs(contest.end_time).format('MMM D, HH:mm')}
                                        </span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 shrink-0">
                                    <Badge
                                        variant={status === 'ongoing' ? 'success' : status === 'upcoming' ? 'default' : 'outline'}
                                        className="text-xs"
                                    >
                                        <StatusIcon size={12} className="mr-1" />
                                        {config.label}
                                    </Badge>
                                    {contest.is_rated && (
                                        <Star size={14} className="text-amber-500 fill-current" />
                                    )}
                                </div>
                            </div>
                        </Link>
                    );
                })}
            </div>
        );
    };

    return (
        <div className="max-w-7xl mx-auto space-y-8 pb-12 animate-in fade-in duration-500">
            {/* Header */}
            <header className="space-y-4">
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div>
                        <h1 className="text-4xl font-black tracking-tight flex items-center gap-3">
                            <Calendar className="text-primary" size={36} />
                            Contest Calendar
                        </h1>
                        <p className="text-muted-foreground mt-1">
                            Track upcoming contests and plan your participation
                        </p>
                    </div>
                </div>
            </header>

            {/* Navigation */}
            <div className="flex items-center justify-between bg-card rounded-xl border p-4 shadow-sm">
                <button
                    onClick={prevMonth}
                    className="p-2 hover:bg-muted rounded-lg transition-colors"
                    aria-label="Previous month"
                >
                    <ChevronLeft size={24} />
                </button>

                <div className="flex items-center gap-4">
                    <h2 className="text-2xl font-bold">
                        {MONTH_NAMES[currentMonth - 1]} {currentYear}
                    </h2>
                    <button
                        onClick={goToToday}
                        className="px-4 py-2 text-sm font-bold bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                    >
                        Today
                    </button>
                </div>

                <button
                    onClick={nextMonth}
                    className="p-2 hover:bg-muted rounded-lg transition-colors"
                    aria-label="Next month"
                >
                    <ChevronRight size={24} />
                </button>
            </div>

            {/* Quick month selection */}
            <div className="flex flex-wrap gap-2">
                {MONTH_NAMES.map((name, index) => (
                    <button
                        key={name}
                        onClick={() => goToMonth(currentYear, index + 1)}
                        className={`px-3 py-1.5 text-xs font-bold rounded-lg transition-colors ${currentMonth === index + 1
                                ? 'bg-primary text-primary-foreground'
                                : 'bg-muted hover:bg-muted/80 text-muted-foreground'
                            }`}
                    >
                        {name.slice(0, 3)}
                    </button>
                ))}
            </div>

            {/* Calendar Grid */}
            {isLoading ? (
                <Skeleton className="h-[400px] w-full rounded-xl" />
            ) : (
                renderCalendarGrid()
            )}

            {/* Contest List */}
            {isLoading ? (
                <div className="space-y-2">
                    {[1, 2, 3].map(i => (
                        <Skeleton key={i} className="h-20 w-full rounded-xl" />
                    ))}
                </div>
            ) : (
                renderContestList()
            )}

            {/* Legend */}
            <div className="bg-card rounded-xl border p-4 shadow-sm">
                <h3 className="text-sm font-bold mb-3">Legend</h3>
                <div className="flex flex-wrap gap-4">
                    <div className="flex items-center gap-2">
                        <div className="w-4 h-4 rounded bg-emerald-500/10 border border-emerald-200" />
                        <span className="text-sm text-muted-foreground">Ongoing</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-4 h-4 rounded bg-blue-500/10 border border-blue-200" />
                        <span className="text-sm text-muted-foreground">Upcoming</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-4 h-4 rounded bg-muted border border-muted" />
                        <span className="text-sm text-muted-foreground">Past</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Star size={14} className="text-amber-500 fill-current" />
                        <span className="text-sm text-muted-foreground">Rated Contest</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
