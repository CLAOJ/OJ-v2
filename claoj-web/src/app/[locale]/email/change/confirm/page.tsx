import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import EmailChangeConfirmContent from './EmailChangeConfirmContent';

export const metadata: Metadata = {
  title: 'Confirm email change',
  robots: { index: false, follow: false },
};

export default async function EmailChangeConfirmPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ token?: string }>;
}) {
  const { locale } = await params;
  const { token } = await searchParams;
  const messages = await getMessages({ locale });

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <EmailChangeConfirmContent token={token ?? ''} />
    </NextIntlClientProvider>
  );
}