import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import LicensePageContent from './LicensePageContent';

async function fetchLicense(key: string) {
  try {
    const api = (await import('@/lib/api')).default;
    const res = await api.get(`/license/${key}`);
    return res.data as { name?: string; text?: string };
  } catch {
    return null;
  }
}

export async function generateMetadata({ params }: { params: Promise<{ key: string }> }): Promise<Metadata> {
  const { key } = await params;
  const license = await fetchLicense(key);
  if (!license) {
    return { title: `License ${key}` };
  }
  return {
    title: `${license.name || key}`,
    description: license.text?.slice(0, 160),
  };
}

export default async function LicensePage({ params }: { params: Promise<{ locale: string; key: string }> }) {
  const { locale } = await params;
  const messages = await getMessages({ locale });

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <LicensePageContent params={params} />
    </NextIntlClientProvider>
  );
}