import { NextRequest, NextResponse } from 'next/server';
import { routing } from '@/navigation';

// POST /api/setlang
// Changes language preference and returns the new locale path
export async function POST(request: NextRequest) {
    try {
        const body = await request.json();
        const { locale, path } = body;

        // Validate locale
        if (!locale || !routing.locales.includes(locale)) {
            return NextResponse.json(
                { error: 'Invalid locale' },
                { status: 400 }
            );
        }

        // localePrefix 'never': keep the same path, switch via cookie.
        const currentPath = path || '/';
        const response = NextResponse.json({
            success: true,
            locale,
            redirect: currentPath,
        });
        response.cookies.set('NEXT_LOCALE', locale, {
            path: '/',
            maxAge: 31536000,
            sameSite: 'lax',
        });
        return response;
    } catch (error) {
        return NextResponse.json(
            { error: 'Invalid request' },
            { status: 400 }
        );
    }
}

// GET /api/setlang - returns current available locales
export async function GET() {
    return NextResponse.json({
        locales: routing.locales,
        defaultLocale: routing.defaultLocale
    });
}
