'use client';

import { useTranslations } from 'next-intl';

interface UserProfile {
    id: number;
    username: string;
}

interface StaffSectionProps {
    users?: { data: UserProfile[] };
    selectedAuthors: number[];
    selectedCurators: number[];
    selectedTesters: number[];
    onUserToggle: (field: 'author_ids' | 'curator_ids' | 'tester_ids', userId: number, checked: boolean) => void;
}

export function StaffSection({ users, selectedAuthors, selectedCurators, selectedTesters, onUserToggle }: StaffSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.staffTitle')}</h3>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('contestForm.authorsLabel')}
                </label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto">
                    {users?.data.map(u => (
                        <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedAuthors.includes(u.id)}
                                onChange={(e) => onUserToggle('author_ids', u.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm truncate">{u.username}</span>
                        </label>
                    ))}
                </div>
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('contestForm.curatorsLabel')}
                </label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto">
                    {users?.data.map(u => (
                        <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedCurators.includes(u.id)}
                                onChange={(e) => onUserToggle('curator_ids', u.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm truncate">{u.username}</span>
                        </label>
                    ))}
                </div>
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('contestForm.testersLabel')}
                </label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto">
                    {users?.data.map(u => (
                        <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedTesters.includes(u.id)}
                                onChange={(e) => onUserToggle('tester_ids', u.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm truncate">{u.username}</span>
                        </label>
                    ))}
                </div>
            </div>
        </div>
    );
}
