import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import SuggestProblemForm from '@/components/problems/SuggestProblemForm';

export const metadata: Metadata = {
  title: 'Suggest a problem | CLAOJ',
  description: 'Propose a new problem for reviewers to consider.',
  robots: { index: false, follow: false },
};

export default async function SuggestProblemNewPage() {
  const locale = await getLocale();
  const messages = await getMessages({ locale });

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <div className="max-w-4xl mx-auto space-y-6 mt-4">
        <SuggestProblemForm />
      </div>
    </NextIntlClientProvider>
  );
}