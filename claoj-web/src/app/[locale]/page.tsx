import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getTranslations } from 'next-intl/server';
import { messagesFor } from '@/i18n/scope';
import type { Metadata } from 'next';
import HomePageContent from './HomePageContent';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'SEO' });

  return {
    title: t('home.title'),
    description: t('home.description'),
    alternates: { canonical: `${SITE_URL}` },
    openGraph: {
      title: t('home.title'),
      description: t('home.description'),
      type: 'website',
      locale: locale === 'vi' ? 'vi_VN' : 'en_US',
      url: `${SITE_URL}`,
      siteName: 'CLAOJ',
    },
  };
}

export default async function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const messages = await messagesFor(locale, ["Admin.banner", "Admin.dashboard.newProblem", "Admin.dashboard.newContest", "Admin.nav.tickets"]);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <HomePageContent />
    </NextIntlClientProvider>
  );
}
