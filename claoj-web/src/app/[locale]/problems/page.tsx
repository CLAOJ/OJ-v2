import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages, getTranslations } from 'next-intl/server';
import type { Metadata } from 'next';
import ProblemsPageContent from './ProblemsPageContent';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'SEO' });
  const title = t('problems.title');
  const description = t('problems.description');

  return {
    title,
    description,
    alternates: { canonical: `${SITE_URL}/problems` },
    openGraph: {
      title,
      description,
      type: 'website',
      locale: locale === 'vi' ? 'vi_VN' : 'en_US',
      url: `${SITE_URL}/problems`,
      siteName: 'CLAOJ',
    },
  };
}

export default async function ProblemsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const messages = await getMessages({ locale });

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <ProblemsPageContent />
    </NextIntlClientProvider>
  );
}
