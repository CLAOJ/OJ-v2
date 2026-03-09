'use client';

import { Link } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { User, LogOut, ChevronDown, Crown } from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';
import { useRef, useEffect } from 'react';
import { useReducedMotion } from 'framer-motion';

interface UserMenuProps {
    username: string;
    isStaff: boolean;
    isOpen: boolean;
    onToggle: () => void;
    onClose: () => void;
    onLogout: () => void;
    onAdminQuickAccess?: () => void;
}

export default function UserMenu({
    username,
    isStaff,
    isOpen,
    onToggle,
    onClose,
    onLogout,
}: UserMenuProps) {
    const userMenuRef = useRef<HTMLDivElement>(null);
    const reduceMotion = useReducedMotion();

    // Click outside to close user menu
    useEffect(() => {
        if (!isOpen) return;

        const handleClickOutside = (e: MouseEvent) => {
            if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
                onClose();
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [isOpen, onClose]);

    const handleLogout = () => {
        onLogout();
        onClose();
    };

    return (
        <div className="relative flex items-center gap-2">
            <button
                onClick={onToggle}
                className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-all text-sm font-bold"
                aria-expanded={isOpen}
                aria-haspopup="true"
            >
                <User size={16} />
                <span className="hidden md:inline">{username}</span>
                <ChevronDown size={14} className={cn("transition-transform", isOpen && "rotate-180")} />
            </button>

            <AnimatePresence>
                {isOpen && (
                    <motion.div
                        ref={userMenuRef}
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: 10 }}
                        transition={{ duration: reduceMotion ? 0 : 0.15 }}
                        className="absolute right-0 top-full mt-2 w-48 bg-card border rounded-lg shadow-xl py-1 z-50 max-h-[calc(100vh-80px)] overflow-y-auto"
                        role="menu"
                        aria-orientation="vertical"
                    >
                        <Link
                            href={`/user/${username}`}
                            className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-muted transition-colors"
                            onClick={onClose}
                            role="menuitem"
                        >
                            <User size={16} />
                            <span>Profile</span>
                        </Link>
                        {isStaff && (
                            <>
                                <Link
                                    href="/admin"
                                    className="flex items-center gap-2 px-4 py-2 text-sm bg-gradient-to-r from-amber-500/10 to-orange-500/10 hover:from-amber-500/20 hover:to-orange-500/20 border-l-2 border-amber-500 transition-colors"
                                    onClick={onClose}
                                    role="menuitem"
                                >
                                    <Crown size={16} className="text-amber-500" />
                                    <span className="font-semibold text-amber-500">Admin Dashboard</span>
                                </Link>
                                <div className="px-4 py-1.5 border-t border-border/50">
                                    <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold mb-1.5">Quick Access</p>
                                    <div className="flex gap-1.5">
                                        <Link
                                            href="/admin/problems/create"
                                            onClick={onClose}
                                            className="flex-1 px-2 py-1 text-[10px] bg-muted hover:bg-muted/80 rounded text-center transition-colors"
                                        >
                                            + Problem
                                        </Link>
                                        <Link
                                            href="/admin/contests/create"
                                            onClick={onClose}
                                            className="flex-1 px-2 py-1 text-[10px] bg-muted hover:bg-muted/80 rounded text-center transition-colors"
                                        >
                                            + Contest
                                        </Link>
                                    </div>
                                </div>
                            </>
                        )}
                        <Link
                            href="/settings"
                            className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-muted transition-colors"
                            onClick={onClose}
                            role="menuitem"
                        >
                            <span>Edit profile</span>
                        </Link>
                        <hr className="my-1 border-border" />
                        <button
                            onClick={handleLogout}
                            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                            role="menuitem"
                        >
                            <LogOut size={16} />
                            <span>Log out</span>
                        </button>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}
