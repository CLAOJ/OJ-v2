import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getTranslations } from 'next-intl/server';
import { messagesFor } from '@/i18n/scope';
import type { Metadata } from 'next';
import ContestsPageContent from './ContestsPageContent';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'SEO' });
  const title = t('contests.title');
  const description = t('contests.description');

  return {
    title,
    description,
    alternates: { canonical: `${SITE_URL}/contests` },
    openGraph: {
      title,
      description,
      type: 'website',
      locale: locale === 'vi' ? 'vi_VN' : 'en_US',
      url: `${SITE_URL}/contests`,
      siteName: 'CLAOJ',
    },
  };
}

export default async function ContestsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const messages = await messagesFor(locale);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <ContestsPageContent />
    </NextIntlClientProvider>
  );
}
