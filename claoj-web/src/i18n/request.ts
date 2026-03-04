import {getRequestConfig} from 'next-intl/server';

export default getRequestConfig(async ({locale}) => {
  // Validate that the incoming `locale` parameter is valid
  const locales: string[] = ['en', 'vi'];

  // Default to 'en' if locale is undefined or invalid
  if (!locale || !locales.includes(locale)) {
    locale = 'en';
  }

  return {
    locale,
    messages: (await import(`../i18n/${locale}.json`)).default
  };
});
