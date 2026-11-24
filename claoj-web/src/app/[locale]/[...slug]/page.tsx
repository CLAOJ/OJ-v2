import { cache } from 'react';
import { cookies } from 'next/headers';
import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import Markdown from '@/components/common/Markdown';
import api from '@/lib/api';
import { normalizeFlatPagePath } from '@/lib/flatpage';

interface FlatPage {
    url: string;
    title: string;
    content: string;
    registration_required: boolean;
}

// Catch-all for v1 flat pages (/about/, /faq/, ...). Static segments in
// app/[locale]/* and their dynamic children win over this route, so it only
// sees paths no real page owns.
async function fetchFlatPage(path: string): Promise<FlatPage | null> {
    try {
        // The browser's cookies must travel with the SSR lookup: the API uses
        // them to tell a signed-in visitor from an anonymous one, which is what
        // decides whether a registration-required page exists.
        const cookieHeader = (await cookies()).toString();
        const res = await api.get<FlatPage>('/page', {
            params: { url: path },
            headers: cookieHeader ? { Cookie: cookieHeader } : undefined,
        });
        return res.data;
    } catch {
        return null;
    }
}

// One backend lookup per request: generateMetadata and the page share it.
const getFlatPage = cache(fetchFlatPage);

function plainText(markdown: string): string {
    return markdown
        .replace(/```[\s\S]*?```/g, ' ')
        .replace(/[#>*_`~[\]]/g, ' ')
        .replace(/\s+/g, ' ')
        .trim();
}

export async function generateMetadata({
    params,
}: {
    params: Promise<{ locale: string; slug: string[] }>;
}): Promise<Metadata> {
    const { slug } = await params;
    const page = await getFlatPage(normalizeFlatPagePath(slug));
    if (!page) return { title: 'Page not found' };
    return {
        title: page.title,
        description: plainText(page.content).slice(0, 160),
    };
}

export default async function FlatPageRoute({
    params,
}: {
    params: Promise<{ locale: string; slug: string[] }>;
}) {
    const { slug } = await params;
    const page = await getFlatPage(normalizeFlatPagePath(slug));
    if (!page) notFound();

    return (
        <article className="mx-auto max-w-3xl">
            <h1 className="mb-6 text-3xl font-bold">{page.title}</h1>
            <Markdown content={page.content} />
        </article>
    );
}