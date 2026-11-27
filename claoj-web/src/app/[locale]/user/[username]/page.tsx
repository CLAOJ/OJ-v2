import { cache } from 'react';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getTranslations } from 'next-intl/server';
import { messagesFor } from '@/i18n/scope';
import type { Metadata } from 'next';
import UserProfilePageContent from './UserProfilePageContent';
import { PersonJsonLd } from '@/components/seo';
import api from '@/lib/api';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch user data for metadata (server-side)
// cache(): generateMetadata and the page share one backend call per request.
// Anonymous on purpose (public data only); the client refetches the
// visitor's own view on mount.
const fetchUser = cache(async (username: string) => {
  try {
    const res = await api.get(`/user/${username}`);
    return res.data;
  } catch (error) {
    return null;
  }
});

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; username: string }>;
}): Promise<Metadata> {
  const { locale, username } = await params;

  // Try to fetch user data
  const user = await fetchUser(username);

  if (user) {
    const t = await getTranslations({ locale, namespace: 'SEO.user' });
    const rating = user.rating || t('noRating');
    const title = `${user.username} (${t('rating')} ${rating})`;
    const description = t('description', { username: user.username, rating, rank: user.rank || '-' });

    return {
      title,
      description,
      alternates: { canonical: `${SITE_URL}/user/${username}` },
      openGraph: {
        title,
        description,
        type: 'profile',
        locale: locale === 'vi' ? 'vi_VN' : 'en_US',
        url: `${SITE_URL}/user/${username}`,
        siteName: 'CLAOJ',
      },
    };
  }

  return {
    title: `User ${username}`,
    description: 'Competitive programmer profile on CLAOJ',
  };
}

export default async function UserProfilePage({
  params,
}: {
  params: Promise<{ locale: string; username: string }>;
}) {
  const { locale } = await params;
  const messages = await messagesFor(locale);
  const { username } = await params;

  // Fetch user data for JSON-LD
  const user = await fetchUser(username);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {user && <PersonJsonLd user={user} />}
      <UserProfilePageContent params={params} initial={user} />
    </NextIntlClientProvider>
  );
}
