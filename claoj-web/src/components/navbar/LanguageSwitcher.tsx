'use client';

import { useLocale, useTranslations } from 'next-intl';
import { usePathname } from '@/navigation';
import { cn } from '@/lib/utils';
import { GB_FLAG_SVG, VI_FLAG_SVG } from '@/lib/flag-icons';

interface LanguageSwitcherProps {
    variant?: 'default' | 'mobile';
}

export default function LanguageSwitcher({ variant = 'default' }: LanguageSwitcherProps) {
    const locale = useLocale();
    const t = useTranslations('Navbar');
    const pathname = usePathname();

    const handleLanguageChange = (newLocale: string) => {
        if (newLocale === locale) return;
        // localePrefix is 'never': locale lives only in the NEXT_LOCALE cookie.
        // Set it, then hard-navigate to the SAME unprefixed path so the server
        // re-renders in the new language. usePathname() drops the query/hash,
        // so re-attach them to preserve filters like /submissions?user=x.
        document.cookie = `NEXT_LOCALE=${newLocale}; path=/; max-age=31536000; SameSite=Lax`;
        const bare = pathname === '/' ? '/' : pathname;
        window.location.assign(`${bare}${window.location.search}${window.location.hash}`);
    };

    if (variant === 'mobile') {
        return (
            <div className="flex items-center justify-between">
                <span className="text-sm font-bold text-foreground">{t('language')}</span>
                <div className="flex items-center gap-2 p-1 rounded bg-muted">
                    <button
                        onClick={() => handleLanguageChange('en')}
                        className={cn(
                            "px-3 py-2 rounded text-sm font-bold",
                            locale === 'en'
                                ? "text-[#009688] bg-white/20"
                                : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                        )}
                        aria-label={t('switchToEnglish')}
                        aria-pressed={locale === 'en'}
                    >
                        EN
                    </button>
                    <span className="text-muted-foreground/50" aria-hidden="true">|</span>
                    <button
                        onClick={() => handleLanguageChange('vi')}
                        className={cn(
                            "px-3 py-2 rounded text-sm font-bold",
                            locale === 'vi'
                                ? "text-[#009688] bg-white/20"
                                : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                        )}
                        aria-label={t('switchToVietnamese')}
                        aria-pressed={locale === 'vi'}
                    >
                        VI
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="flex items-center gap-1 border-l border-border pl-3">
            <button
                onClick={() => handleLanguageChange('en')}
                className={cn(
                    "flex items-center gap-1 px-2 py-1 rounded text-xs font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
                    locale === 'en'
                        ? "bg-[#009688] text-white"
                        : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                )}
                aria-label={t('switchToEnglish')}
                aria-pressed={locale === 'en'}
            >
                <img src="/static/icons/gb_flag.svg" alt="" className="w-4 h-3 object-cover" onError={(e) => (e.target as HTMLImageElement).src = GB_FLAG_SVG} />
                EN
            </button>
            <button
                onClick={() => handleLanguageChange('vi')}
                className={cn(
                    "flex items-center gap-1 px-2 py-1 rounded text-xs font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
                    locale === 'vi'
                        ? "bg-[#009688] text-white"
                        : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                )}
                aria-label={t('switchToVietnamese')}
                aria-pressed={locale === 'vi'}
            >
                <img src="/static/icons/vi_flag.svg" alt="" className="w-4 h-3 object-cover" onError={(e) => (e.target as HTMLImageElement).src = VI_FLAG_SVG} />
                VI
            </button>
        </div>
    );
}
