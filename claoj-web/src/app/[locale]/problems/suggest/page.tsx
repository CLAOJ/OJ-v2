import { NextIntlClientProvider } from 'next-intl';
import { getLocale } from 'next-intl/server';
import { messagesFor } from '@/i18n/scope';
import type { Metadata } from 'next';
import SuggestProblemsContent from './SuggestProblemsContent';

export const metadata: Metadata = {
  title: 'Suggested Problems',
  description: 'Problems suggested by users and still awaiting review.',
  // Staff-visible list: keep it out of search indexes.
  robots: { index: false, follow: false },
};

export default async function SuggestProblemsPage() {
  const locale = await getLocale();
  const messages = await messagesFor(locale);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <SuggestProblemsContent />
    </NextIntlClientProvider>
  );
}
