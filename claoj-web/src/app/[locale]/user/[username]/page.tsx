import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import UserProfilePageContent from './UserProfilePageContent';
import { PersonJsonLd } from '@/components/seo';
import api from '@/lib/api';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch user data for metadata (server-side)
async function fetchUser(username: string) {
  try {
    const res = await api.get(`http://localhost:8080/api/v2/user/${username}`);
    return res.data;
  } catch (error) {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; username: string }>;
}): Promise<Metadata> {
  const { locale, username } = await params;

  // Try to fetch user data
  const user = await fetchUser(username);

  if (user) {
    const title = `${user.username} - ${user.rating || 'N/A'} rating | CLAOJ`;
    const description = `Profile of ${user.username} - Rating: ${user.rating || 'N/A'}, Rank: ${user.rank || 'N/A'}`;

    return {
      title,
      description,
      openGraph: {
        title,
        description,
        type: 'profile',
        locale: locale === 'vi' ? 'vi_VN' : 'en_US',
        url: `${SITE_URL}/${locale}/user/${username}`,
        siteName: 'CLAOJ',
      },
    };
  }

  return {
    title: `User ${username} | CLAOJ`,
    description: 'Competitive programmer profile on CLAOJ',
  };
}

export default async function UserProfilePage({
  params,
}: {
  params: Promise<{ locale: string; username: string }>;
}) {
  const { locale } = await params;
  const messages = await getMessages({ locale });
  const { username } = await params;

  // Fetch user data for JSON-LD
  const user = await fetchUser(username);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {user && <PersonJsonLd user={user} />}
      <UserProfilePageContent params={params} />
    </NextIntlClientProvider>
  );
}
