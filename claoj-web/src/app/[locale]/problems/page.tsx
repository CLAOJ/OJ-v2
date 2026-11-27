import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages, getTranslations } from 'next-intl/server';
import type { Metadata } from 'next';
import { cookies } from 'next/headers';
import api from '@/lib/api';
import { defaultProblemListParams, TRENDING_LIMIT } from '@/lib/problemFilters';
import type { PaginatedList, Problem, ProblemTaxonomy } from '@/types';
import ProblemsPageContent, { type ProblemsInitialData } from './ProblemsPageContent';

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

// Fetch the first view on the server (one hop over the internal network) so the
// list ships in the HTML instead of waiting for JS + four browser round trips.
// Cookies are forwarded so `is_solved` matches the visitor; any failure just
// falls back to the client-side fetch.
async function fetchInitial(): Promise<ProblemsInitialData> {
  const cookie = (await cookies()).toString();
  const opts = { headers: cookie ? { Cookie: cookie } : undefined };
  const get = <T,>(url: string) => api.get<T>(url, opts).then(r => r.data).catch(() => undefined);
  const [problems, groups, types, hot] = await Promise.all([
    get<PaginatedList<Problem>>(`/problems?${defaultProblemListParams()}`),
    get<PaginatedList<ProblemTaxonomy>>('/problem-groups'),
    get<PaginatedList<ProblemTaxonomy>>('/problem-types'),
    get<PaginatedList<Problem>>(`/problems?sort=user_count&order=desc&page_size=${TRENDING_LIMIT}`),
  ]);
  return {
    problems,
    groups: groups?.data,
    types: types?.data,
    hot: hot?.data.slice(0, TRENDING_LIMIT),
  };
}

export default async function ProblemsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const [messages, initial] = await Promise.all([getMessages({ locale }), fetchInitial()]);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <ProblemsPageContent initial={initial} />
    </NextIntlClientProvider>
  );
}
