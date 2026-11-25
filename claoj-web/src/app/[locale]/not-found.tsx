import { getTranslations } from 'next-intl/server';
import { Link } from '@/navigation';

// Rendered inside the locale layout, so the site's nav and footer stay put and
// the visitor can keep going instead of hitting Next's bare English page.
export default async function NotFound() {
    const t = await getTranslations('Common');

    return (
        <section className="mx-auto max-w-xl py-16 text-center">
            <h1 className="text-3xl font-bold text-balance">{t('pageNotFound')}</h1>
            <p className="mt-3 text-muted-foreground text-pretty">{t('pageNotFoundDesc')}</p>
            <div className="mt-8 flex flex-wrap justify-center gap-3">
                <Link
                    href="/problems"
                    className="inline-flex h-10 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                >
                    {t('browseProblems')}
                </Link>
                <Link
                    href="/"
                    className="inline-flex h-10 items-center rounded-md border px-4 text-sm font-medium hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                >
                    {t('goHome')}
                </Link>
            </div>
        </section>
    );
}
