import createMiddleware from 'next-intl/middleware';
import { routing } from './src/navigation';

export default createMiddleware(routing);

export const config = {
  // Match all paths except for:
  // - API routes
  // - Static routes
  // - Next.js internal paths (_next, _vercel)
  // - Files with extensions (e.g., favicon.ico)
  matcher: ['/((?!api|static|_next|_vercel|.*\\.).*)']
};
