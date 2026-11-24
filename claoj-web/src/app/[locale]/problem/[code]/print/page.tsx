import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import api from '@/lib/api';
import type { ProblemDetail } from '@/types';
import ProblemPrintView from './ProblemPrintView';

async function fetchProblem(code: string): Promise<ProblemDetail | null> {
    try {
        const res = await api.get<ProblemDetail>(`/problem/${code}`);
        return res.data;
    } catch {
        return null;
    }
}

export async function generateMetadata({
    params,
}: {
    params: Promise<{ locale: string; code: string }>;
}): Promise<Metadata> {
    const { code } = await params;
    const problem = await fetchProblem(code);
    return {
        title: problem ? `${problem.code}: ${problem.name}` : `Problem ${code}`,
        // A print view is a duplicate of the problem page; keep it out of search.
        robots: { index: false, follow: false },
        alternates: { canonical: `/problem/${code}` },
    };
}

export default async function ProblemPrintPage({
    params,
    searchParams,
}: {
    params: Promise<{ locale: string; code: string }>;
    searchParams: Promise<{ print?: string }>;
}) {
    const { code } = await params;
    const { print } = await searchParams;
    const problem = await fetchProblem(code);
    if (!problem) notFound();

    return <ProblemPrintView problem={problem} autoPrint={print === '1'} />;
}