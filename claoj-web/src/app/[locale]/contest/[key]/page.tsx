import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages, getTranslations } from 'next-intl/server';
import type { Metadata } from 'next';
import ContestPageContent from './ContestPageContent';
import { ContestJsonLd } from '@/components/seo';
import api from '@/lib/api';
import { contestFormatLabel } from '@/lib/utils';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch contest data for metadata (server-side)
async function fetchContest(key: string) {
  try {
    const res = await api.get(`/contest/${key}`);
    return res.data;
  } catch (error) {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; key: string }>;
}): Promise<Metadata> {
  const { locale, key } = await params;

  // Try to fetch contest data
  const contest = await fetchContest(key);

  if (contest) {
    // start_time/end_time arrive as ISO strings; comparing a Date with a string is always false.
    const now = Date.now();
    const isRunning = now > Date.parse(contest.start_time) && now < Date.parse(contest.end_time);
    const isPast = now > Date.parse(contest.end_time);
    const t = await getTranslations({ locale, namespace: 'SEO.contest' });
    const status = t(isPast ? 'ended' : isRunning ? 'ongoing' : 'upcoming');
    const format = contestFormatLabel(contest.format);

    const title = `${contest.name} - ${format}`;
    const description = contest.summary || t('description', { name: contest.name, format, status });

    return {
      title,
      description,
      alternates: { canonical: `${SITE_URL}/contest/${key}` },
      openGraph: {
        title,
        description,
        type: 'website',
        locale: locale === 'vi' ? 'vi_VN' : 'en_US',
        url: `${SITE_URL}/contest/${key}`,
        siteName: 'CLAOJ',
      },
    };
  }

  return {
    title: `Contest ${key}`,
    description: 'Competitive programming contest on CLAOJ',
  };
}

export default async function ContestPage({
  params,
}: {
  params: Promise<{ locale: string; key: string }>;
}) {
  const { locale } = await params;
  const messages = await getMessages({ locale });
  const { key } = await params;

  // Fetch contest data for JSON-LD
  const contest = await fetchContest(key);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {contest && <ContestJsonLd contest={contest} />}
      <ContestPageContent params={params} />
    </NextIntlClientProvider>
  );
}
