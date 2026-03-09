import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import ContestPageContent from './ContestPageContent';
import { ContestJsonLd } from '@/components/seo';
import api from '@/lib/api';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch contest data for metadata (server-side)
async function fetchContest(key: string) {
  try {
    const res = await api.get(`http://localhost:8080/api/v2/contest/${key}`);
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
    const isRunning = new Date() > contest.start_time && new Date() < contest.end_time;
    const isPast = new Date() > contest.end_time;
    const status = isPast ? 'Ended' : isRunning ? 'Running' : 'Upcoming';

    const title = `${contest.name} - ${contest.format} ${status} | CLAOJ`;
    const description = `${contest.summary || contest.description || ''} | ${contest.format} format, ${isRunning ? 'Currently Running' : status}`;

    return {
      title,
      description,
      openGraph: {
        title,
        description,
        type: 'website',
        locale: locale === 'vi' ? 'vi_VN' : 'en_US',
        url: `${SITE_URL}/${locale}/contests/${key}`,
        siteName: 'CLAOJ',
      },
    };
  }

  return {
    title: `Contest ${key} | CLAOJ`,
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
