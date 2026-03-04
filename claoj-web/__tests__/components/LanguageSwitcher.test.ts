import { describe, it, expect } from '@jest/globals';

describe('Language Switching', () => {
    describe('handleLanguageChange logic', () => {
        it('should switch from English (/) to Vietnamese (/vi)', () => {
            const pathname = '/';
            const targetLang = 'vi';
            const currentLocale = pathname.split('/')[1];
            const effectiveCurrent = currentLocale === 'vi' ? 'vi' : 'en';

            let newPath: string;
            if (targetLang === 'en') {
                newPath = effectiveCurrent === 'en' ? pathname : pathname.replace(/^\/vi/, '');
            } else {
                newPath = effectiveCurrent === 'en' ? `/${targetLang}${pathname}` : pathname.replace(/^\/vi/, `/${targetLang}`);
            }

            // With localePrefix 'as-needed', Vietnamese gets prefix
            expect(newPath).toMatch(/\/vi/);
        });

        it('should switch from Vietnamese (/vi) to English (/)', () => {
            const pathname = '/vi';
            const targetLang = 'en';
            const currentLocale = pathname.split('/')[1];
            const effectiveCurrent = currentLocale === 'vi' ? 'vi' : 'en';

            let newPath: string;
            if (targetLang === 'en') {
                newPath = effectiveCurrent === 'en' ? pathname : pathname.replace(/^\/vi/, '');
            } else {
                newPath = effectiveCurrent === 'en' ? `/${targetLang}${pathname}` : pathname.replace(/^\/vi/, `/${targetLang}`);
            }

            // English is default, so path should be empty or /
            expect(newPath === '/' || newPath === '').toBe(true);
        });

        it('should switch from /vi/problems to /problems', () => {
            const pathname = '/vi/problems';
            const targetLang = 'en';
            const currentLocale = pathname.split('/')[1];
            const effectiveCurrent = currentLocale === 'vi' ? 'vi' : 'en';

            let newPath: string;
            if (targetLang === 'en') {
                newPath = effectiveCurrent === 'en' ? pathname : pathname.replace(/^\/vi/, '');
            } else {
                newPath = effectiveCurrent === 'en' ? `/${targetLang}${pathname}` : pathname.replace(/^\/vi/, `/${targetLang}`);
            }

            expect(newPath).toBe('/problems');
        });

        it('should switch from /problems to /vi/problems', () => {
            const pathname = '/problems';
            const targetLang = 'vi';
            const currentLocale = pathname.split('/')[1];
            const effectiveCurrent = currentLocale === 'vi' ? 'vi' : 'en';

            let newPath: string;
            if (targetLang === 'en') {
                newPath = effectiveCurrent === 'en' ? pathname : pathname.replace(/^\/vi/, '');
            } else {
                newPath = effectiveCurrent === 'en' ? `/${targetLang}${pathname}` : pathname.replace(/^\/vi/, `/${targetLang}`);
            }

            expect(newPath).toBe('/vi/problems');
        });
    });
});
