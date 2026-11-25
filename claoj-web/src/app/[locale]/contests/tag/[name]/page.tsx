import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import ContestsPageContent from '../../ContestsPageContent';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// v1 /contests/tag/<name> (ContestTagDetail): the contest list narrowed to one
// tag. Same segment name as the other dynamic routes under /contests? There are
// none — `tag` is a static segment beside `calendar`.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; name: string }>;
}): Promise<Metadata> {
  const { locale, name } = await params;
  const tag = decodeURIComponent(name);
  const title = `Contests tagged ${tag} | CLAOJ`;
  const description = `Competitive programming contests tagged ${tag} on CLAOJ.`;

  return {
    title,
    description,
    alternates: { canonical: `${SITE_URL}/contests/tag/${name}` },
    openGraph: {
      title,
      description,
      type: 'website',
      locale: locale === 'vi' ? 'vi_VN' : 'en_US',
      url: `${SITE_URL}/contests/tag/${name}`,
      siteName: 'CLAOJ',
    },
  };
}

export default async function ContestTagPage({
  params,
}: {
  params: Promise<{ locale: string; name: string }>;
}) {
  const { locale, name } = await params;
  const messages = await getMessages({ locale });
  const tag = decodeURIComponent(name);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <ContestsPageContent initialTag={tag} />
    </NextIntlClientProvider>
  );
}