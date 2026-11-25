'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { SlidersHorizontal } from 'lucide-react';
import api, { type ApiError } from '@/lib/api';
import { Select } from '@/components/ui/Select';
import { ACE_THEMES, SITE_THEMES, timezones, withCurrent } from '@/lib/profileChoices';

interface Prefs {
    timezone: string;
    language_id: number;
    ace_theme: string;
    site_theme: string;
}

// Timezone, default submission language and themes (v1's profile edit page).
// Loads the current values from /user/me; the editor mounts only once they
// are known so the selects never show a guessed default.
export function PreferencesSection() {
    const { data } = useQuery({
        queryKey: ['user-me-prefs'],
        queryFn: async () => (await api.get<{ profile: Prefs }>('/user/me')).data.profile,
    });
    if (!data) return null;
    return <PreferencesEditor initial={data} />;
}

function PreferencesEditor({ initial }: { initial: Prefs }) {
    const t = useTranslations('Settings');
    const queryClient = useQueryClient();
    const [prefs, setPrefs] = useState<Prefs>(initial);
    const dirty = (Object.keys(prefs) as (keyof Prefs)[]).some(k => prefs[k] !== initial[k]);

    const { data: languages = [] } = useQuery({
        queryKey: ['languages-public'],
        queryFn: async () => (await api.get<{ data: { id: number; name: string }[] }>('/languages')).data.data,
    });

    const save = useMutation({
        mutationFn: () => api.patch('/user/me', prefs),
        onSuccess: () => {
            toast.success(t('updateSuccess'));
            queryClient.invalidateQueries({ queryKey: ['user-me-prefs'] });
        },
        onError: (err) => toast.error((err as ApiError).response?.data?.error || t('updateProfileError')),
    });

    const field = (id: string, label: string, control: React.ReactNode) => (
        <div className="space-y-2">
            <label htmlFor={id} className="text-sm font-medium">{label}</label>
            {control}
        </div>
    );

    return (
        <section className="space-y-6">
            <div className="flex items-center gap-2 text-primary font-medium">
                <SlidersHorizontal size={18} />
                {t('preferences')}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {field('pref-timezone', t('timezone'), (
                    <Select id="pref-timezone" value={prefs.timezone} onChange={e => setPrefs({ ...prefs, timezone: e.target.value })}>
                        {withCurrent(timezones(), prefs.timezone).map(tz => <option key={tz} value={tz}>{tz}</option>)}
                    </Select>
                ))}
                {field('pref-language', t('language'), (
                    <Select id="pref-language" value={prefs.language_id} onChange={e => setPrefs({ ...prefs, language_id: Number(e.target.value) })}>
                        {!languages.some(l => l.id === prefs.language_id) && <option value={prefs.language_id}>#{prefs.language_id}</option>}
                        {languages.map(l => <option key={l.id} value={l.id}>{l.name}</option>)}
                    </Select>
                ))}
                {field('pref-ace-theme', t('aceTheme'), (
                    <Select id="pref-ace-theme" value={prefs.ace_theme} onChange={e => setPrefs({ ...prefs, ace_theme: e.target.value })}>
                        {ACE_THEMES.map(([v, label]) => <option key={v} value={v}>{v === 'auto' ? t('aceThemeAuto') : label}</option>)}
                    </Select>
                ))}
                {field('pref-site-theme', t('theme'), (
                    <Select id="pref-site-theme" value={prefs.site_theme} onChange={e => setPrefs({ ...prefs, site_theme: e.target.value })}>
                        {SITE_THEMES.map(v => (
                            <option key={v} value={v}>
                                {v === 'auto' ? t('siteThemeAuto') : v === 'light' ? t('siteThemeLight') : t('siteThemeDark')}
                            </option>
                        ))}
                    </Select>
                ))}
            </div>
            <button
                type="button"
                onClick={() => save.mutate()}
                disabled={!dirty || save.isPending}
                className="px-8 h-12 rounded-xl bg-primary text-primary-foreground font-medium hover:opacity-90 disabled:opacity-50 transition-all"
            >
                {t('save')}
            </button>
        </section>
    );
}
