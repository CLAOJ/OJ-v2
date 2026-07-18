import createMiddleware from 'next-intl/middleware';
import { routing } from './navigation';

export default createMiddleware(routing);

export const config = {
    matcher: ['/((?!api|static|_next|_vercel|.*\\..*).*)']
};
