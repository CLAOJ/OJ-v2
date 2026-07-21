'use client';

import React, { useState } from 'react';
import { useAuth } from '@/components/providers/AuthProvider';
import { AdminSidebar } from './AdminSidebar';
import { AdminQuickAccessButton } from './AdminQuickAccessButton';
import { AdminWelcomeBanner } from './AdminWelcomeBanner';

interface AdminAccessWrapperProps {
    children: React.ReactNode;
    showWelcomeBanner?: boolean;
}

export function AdminAccessWrapper({ children, showWelcomeBanner = false }: AdminAccessWrapperProps) {
    const { user } = useAuth();
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [bannerDismissed, setBannerDismissed] = useState(false);

    // Superuser-inclusive: admins bypass the staff gate (see backend route gate).
    const isAdmin = user?.is_staff || user?.is_admin;

    return (
        <>
            {isAdmin && (
                <>
                    <AdminSidebar isOpen={sidebarOpen} onClose={() => setSidebarOpen(false)} />
                    <AdminQuickAccessButton onClick={() => setSidebarOpen(true)} />
                </>
            )}

            <div className="relative">
                {isAdmin && showWelcomeBanner && !bannerDismissed && (
                    <AdminWelcomeBanner onDismiss={() => setBannerDismissed(true)} />
                )}
                {children}
            </div>
        </>
    );
}

export default AdminAccessWrapper;
