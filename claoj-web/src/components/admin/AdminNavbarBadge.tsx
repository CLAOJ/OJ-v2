'use client';

import React from 'react';
import { motion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import { Shield } from 'lucide-react';
import { cn } from '@/lib/utils';

interface AdminNavbarBadgeProps {
    onClick?: () => void;
}

export function AdminNavbarBadge({ onClick }: AdminNavbarBadgeProps) {
    const { user } = useAuth();

    if (!user?.is_staff) return null;

    return (
        <motion.button
            onClick={onClick}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            className={cn(
                'flex items-center gap-2 px-3 py-1.5 rounded-full',
                'bg-gradient-to-r from-amber-500/20 to-orange-500/20',
                'border border-amber-500/30',
                'text-amber-400 text-xs font-bold uppercase tracking-wider',
                'hover:from-amber-500/30 hover:to-orange-500/30 hover:border-amber-500/50',
                'transition-all duration-200'
            )}
        >
            <Shield className="w-3.5 h-3.5" />
            <span>Admin</span>
        </motion.button>
    );
}
