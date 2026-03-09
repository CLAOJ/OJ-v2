'use client';

import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import { Settings } from 'lucide-react';
import { cn } from '@/lib/utils';

interface AdminQuickAccessButtonProps {
    onClick: () => void;
}

export function AdminQuickAccessButton({ onClick }: AdminQuickAccessButtonProps) {
    const { user } = useAuth();
    const [isHovered, setIsHovered] = useState(false);
    const [showPulse, setShowPulse] = useState(true);

    if (!user?.is_staff) return null;

    return (
        <motion.button
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ type: 'spring', damping: 20, stiffness: 300, delay: 0.5 }}
            onClick={onClick}
            onMouseEnter={() => {
                setIsHovered(true);
                setShowPulse(false);
            }}
            onMouseLeave={() => setIsHovered(false)}
            className={cn(
                'fixed left-4 top-24 z-30 group',
                'flex items-center gap-2 pl-3 pr-4 py-2.5 rounded-full',
                'bg-slate-950/90 backdrop-blur-md border border-amber-500/30',
                'shadow-lg shadow-amber-500/10 hover:shadow-amber-500/20',
                'hover:border-amber-500/50 transition-all duration-300'
            )}
        >
            {/* Pulse effect */}
            {showPulse && (
                <span className="absolute inset-0 rounded-full bg-amber-500/20 animate-ping" />
            )}

            {/* Icon container */}
            <div className="relative">
                <motion.div
                    animate={{ rotate: isHovered ? 180 : 0 }}
                    transition={{ duration: 0.3 }}
                    className="w-8 h-8 rounded-full bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center"
                >
                    <Settings className="w-4 h-4 text-slate-950" />
                </motion.div>
                <span className="absolute -top-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-slate-950" />
            </div>

            {/* Label */}
            <motion.span
                animate={{ x: isHovered ? 2 : 0 }}
                className="text-sm font-semibold text-amber-400"
            >
                Admin
            </motion.span>
        </motion.button>
    );
}
