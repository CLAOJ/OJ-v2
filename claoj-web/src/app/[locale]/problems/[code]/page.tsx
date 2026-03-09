import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import ProblemPageContent from './ProblemPageContent';
import { ProblemJsonLd } from '@/components/seo';
import api from '@/lib/api';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch problem data for metadata (server-side)
async function fetchProblem(code: string) {
  try {
    const res = await api.get(`http://localhost:8080/api/v2/problem/${code}`);
    return res.data;
  } catch (error) {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; code: string }>;
}): Promise<Metadata> {
  const { locale, code } = await params;

  // Try to fetch problem data
  const problem = await fetchProblem(code);

  if (problem) {
    const title = `${problem.code}: ${problem.name}`;
    const description = `${problem.points} points | ${Math.round(problem.ac_rate)}% AC rate`;

    return {
      title,
      description,
      openGraph: {
        title,
        description,
        type: 'website',
        locale: locale === 'vi' ? 'vi_VN' : 'en_US',
        url: `${SITE_URL}/${locale}/problems/${code}`,
        siteName: 'CLAOJ',
      },
    };
  }

  return {
    title: `Problem ${code} | CLAOJ`,
    description: 'Solve competitive programming problems on CLAOJ',
  };
}

export default async function ProblemPage({
  params,
}: {
  params: Promise<{ locale: string; code: string }>;
}) {
  const { locale } = await params;
  const messages = await getMessages({ locale });
  const { code } = await params;

  // Fetch problem data for JSON-LD
  const problem = await fetchProblem(code);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {problem && <ProblemJsonLd problem={problem} />}
      <ProblemPageContent params={params} />
    </NextIntlClientProvider>
  );
}
