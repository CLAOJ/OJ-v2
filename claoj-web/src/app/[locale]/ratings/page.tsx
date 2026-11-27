import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getTranslations } from 'next-intl/server';
import { messagesFor } from '@/i18n/scope';
import type { Metadata } from 'next';
import RatingsPageContent from './RatingsPageContent';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'SEO' });
  const title = t('ratings.title');
  const description = t('ratings.description');

  return {
    title,
    description,
    alternates: { canonical: `${SITE_URL}/ratings` },
    openGraph: {
      title,
      description,
      type: 'website',
      locale: locale === 'vi' ? 'vi_VN' : 'en_US',
      url: `${SITE_URL}/ratings`,
      siteName: 'CLAOJ',
    },
  };
}

export default async function RatingsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const messages = await messagesFor(locale);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <RatingsPageContent />
    </NextIntlClientProvider>
  );
}
