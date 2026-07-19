'use client';

import { useAuth } from '@/components/providers/AuthProvider';
import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { User, Shield, Bell, Key, Globe, Download } from 'lucide-react';
import { cn } from '@/lib/utils';
import ProfileSettingsTab from '@/components/settings/ProfileSettingsTab';
import AccountSettingsTab from '@/components/settings/AccountSettingsTab';
import OAuthSettingsTab from '@/components/settings/OAuthSettingsTab';
import NotificationSettingsTab from '@/components/settings/NotificationSettingsTab';
import APITokenSettingsTab from '@/components/settings/APITokenSettingsTab';
import DataExportSettingsTab from '@/components/settings/DataExportSettingsTab';
import WebAuthnSettings from '@/components/settings/WebAuthnSettings';

type ActiveTab = 'profile' | 'account' | 'oauth' | 'notifications' | 'api-token' | 'data-export' | 'webauthn';

export default function SettingsPage() {
    const { user } = useAuth();
    const t = useTranslations('Settings');
    const [activeTab, setActiveTab] = useState<ActiveTab>('profile');

    return (
        <div className="max-w-4xl mx-auto space-y-8 animate-in fade-in duration-500">
            <header>
                <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>
                <p className="text-muted-foreground mt-1">{t('subtitle')}</p>
            </header>

            <div className="flex flex-col md:flex-row gap-8">
                {/* Sidebar Tabs */}
                <aside className="w-full md:w-64 space-y-1">
                    <TabButton icon={User} label={t('profile')} active={activeTab === 'profile'} onClick={() => setActiveTab('profile')} />
                    <TabButton icon={Shield} label={t('account')} active={activeTab === 'account'} onClick={() => setActiveTab('account')} />
                    <TabButton icon={Globe} label={t('oauth')} active={activeTab === 'oauth'} onClick={() => setActiveTab('oauth')} />
                    <TabButton icon={Bell} label={t('notifications')} active={activeTab === 'notifications'} onClick={() => setActiveTab('notifications')} />
                    <TabButton icon={Key} label={t('apiToken')} active={activeTab === 'api-token'} onClick={() => setActiveTab('api-token')} />
                    <TabButton icon={Download} label={t('dataExport')} active={activeTab === 'data-export'} onClick={() => setActiveTab('data-export')} />
                    <TabButton icon={Shield} label={t('passkey')} active={activeTab === 'webauthn'} onClick={() => setActiveTab('webauthn')} />
                </aside>

                {/* Content Area */}
                <div className="flex-grow p-8 rounded-3xl border bg-card shadow-sm">
                    {activeTab === 'profile' && <ProfileSettingsTab />}
                    {activeTab === 'account' && <AccountSettingsTab />}
                    {activeTab === 'oauth' && <OAuthSettingsTab />}
                    {activeTab === 'notifications' && <NotificationSettingsTab />}
                    {activeTab === 'api-token' && <APITokenSettingsTab />}
                    {activeTab === 'data-export' && <DataExportSettingsTab />}
                    {activeTab === 'webauthn' && <WebAuthnSettings />}
                </div>
            </div>
        </div>
    );
}

function TabButton({ icon: Icon, label, active, onClick }: { icon: React.ComponentType<{ size: number }>; label: string; active: boolean; onClick: () => void }) {
    return (
        <button
            onClick={onClick}
            className={cn(
                "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                active
                    ? "bg-primary/10 text-primary"
                    : "hover:bg-muted text-muted-foreground"
            )}
        >
            <Icon size={18} />
            {label}
        </button>
    );
}
