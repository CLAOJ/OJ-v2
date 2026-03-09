'use client';

import React, { useState } from 'react';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import { Link } from '@/navigation';
import {
    Shield,
    X,
    ChevronRight,
    Crown,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ADMIN_SECTIONS } from './constants';

interface AdminSidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

export function AdminSidebar({ isOpen, onClose }: AdminSidebarProps) {
    const { user } = useAuth();
    const reduceMotion = useReducedMotion();
    const [hoveredItem, setHoveredItem] = useState<string | null>(null);

    if (!user?.is_staff) return null;

    return (
        <AnimatePresence>
            {isOpen && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: reduceMotion ? 0 : 0.2 }}
                        className="fixed inset-0 bg-black/60 backdrop-blur-sm z-40"
                        onClick={onClose}
                    />

                    {/* Sidebar */}
                    <motion.aside
                        initial={{ x: '-100%', opacity: 0.8 }}
                        animate={{ x: 0, opacity: 1 }}
                        exit={{ x: '-100%', opacity: 0.8 }}
                        transition={{
                            type: 'spring',
                            damping: 30,
                            stiffness: 300,
                            mass: 0.8,
                        }}
                        className="fixed left-0 top-0 h-full w-80 z-50"
                    >
                        <div className="h-full bg-slate-950 border-r border-slate-800/50 flex flex-col overflow-hidden">
                            {/* Header */}
                            <div className="relative overflow-hidden">
                                <div className="absolute inset-0 bg-gradient-to-br from-amber-500/10 via-transparent to-transparent" />
                                <div className="relative p-6 border-b border-slate-800/50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center gap-3">
                                            <div className="relative">
                                                <div className="absolute inset-0 bg-amber-500/30 blur-xl rounded-full" />
                                                <div className="relative w-12 h-12 rounded-xl bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center shadow-lg shadow-amber-500/25">
                                                    <Crown className="w-6 h-6 text-slate-950" />
                                                </div>
                                            </div>
                                            <div>
                                                <h2 className="text-lg font-bold text-slate-100">
                                                    Admin Panel
                                                </h2>
                                                <p className="text-xs text-slate-400 font-medium">
                                                    Command Center
                                                </p>
                                            </div>
                                        </div>
                                        <button
                                            onClick={onClose}
                                            className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
                                        >
                                            <X className="w-5 h-5" />
                                        </button>
                                    </div>
                                </div>
                            </div>

                            {/* Admin Info */}
                            <div className="px-4 py-3 border-b border-slate-800/50">
                                <div className="flex items-center gap-3 p-3 rounded-lg bg-slate-900/50 border border-slate-800">
                                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold">
                                        {user.username[0].toUpperCase()}
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <p className="text-sm font-semibold text-slate-200 truncate">
                                            {user.username}
                                        </p>
                                        <div className="flex items-center gap-1.5">
                                            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                                            <span className="text-xs text-slate-400">
                                                {user.is_admin ? 'Super Admin' : 'Staff'}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {/* Navigation */}
                            <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
                                {ADMIN_SECTIONS.map((section, index) => (
                                    <motion.div
                                        key={section.id}
                                        initial={{ opacity: 0, x: -20 }}
                                        animate={{ opacity: 1, x: 0 }}
                                        transition={{ delay: index * 0.03 }}
                                    >
                                        <Link
                                            href={section.href}
                                            onClick={onClose}
                                            onMouseEnter={() => setHoveredItem(section.id)}
                                            onMouseLeave={() => setHoveredItem(null)}
                                            className={cn(
                                                'group flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200',
                                                'hover:bg-slate-800/50 relative overflow-hidden'
                                            )}
                                        >
                                            {/* Hover gradient */}
                                            <AnimatePresence>
                                                {hoveredItem === section.id && (
                                                    <motion.div
                                                        layoutId="hoverGradient"
                                                        className={cn(
                                                            'absolute inset-0 bg-gradient-to-r opacity-10',
                                                            section.color
                                                        )}
                                                        initial={{ opacity: 0 }}
                                                        animate={{ opacity: 0.1 }}
                                                        exit={{ opacity: 0 }}
                                                    />
                                                )}
                                            </AnimatePresence>

                                            {/* Icon */}
                                            <div
                                                className={cn(
                                                    'relative w-9 h-9 rounded-lg flex items-center justify-center',
                                                    'bg-gradient-to-br text-white shadow-lg',
                                                    section.color,
                                                    'group-hover:scale-110 transition-transform duration-200'
                                                )}
                                            >
                                                <section.icon className="w-4.5 h-4.5" />
                                            </div>

                                            {/* Label */}
                                            <span className="relative flex-1 text-sm font-medium text-slate-300 group-hover:text-slate-100">
                                                {section.label}
                                            </span>

                                            {/* Badge */}
                                            {section.badge && (
                                                <span className="relative text-[10px] font-bold px-2 py-0.5 rounded-full bg-slate-800 text-slate-400">
                                                    {section.badge}
                                                </span>
                                            )}

                                            {/* Arrow */}
                                            <ChevronRight className="relative w-4 h-4 text-slate-600 group-hover:text-slate-400 group-hover:translate-x-0.5 transition-all" />
                                        </Link>
                                    </motion.div>
                                ))}
                            </nav>

                            {/* Footer */}
                            <div className="p-4 border-t border-slate-800/50">
                                <div className="flex items-center gap-2 text-xs text-slate-500">
                                    <Shield className="w-3.5 h-3.5" />
                                    <span>Secure Admin Session</span>
                                </div>
                            </div>
                        </div>
                    </motion.aside>
                </>
            )}
        </AnimatePresence>
    );
}
