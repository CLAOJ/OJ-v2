'use client';

import { useEffect, useState } from 'react';
import { randomProblemApi } from '@/lib/api';
import { useRouter } from '@/navigation';
import { Skeleton } from '@/components/ui/Skeleton';
import { Shuffle } from 'lucide-react';

export default function RandomProblemPage() {
    const [error, setError] = useState<string | null>(null);
    const router = useRouter();

    useEffect(() => {
        randomProblemApi.getRandomProblem()
            .then((res) => {
                router.push(`/problems/${res.data.code}`);
            })
            .catch((err) => {
                console.error('Failed to get random problem:', err);
                setError('Failed to load random problem. Please try again.');
            });
    }, []);

    if (error) {
        return (
            <div className="max-w-md mx-auto mt-20 p-6 text-center">
                <Shuffle size={48} className="mx-auto mb-4 text-destructive" />
                <h1 className="text-2xl font-bold mb-2">Error</h1>
                <p className="text-muted-foreground mb-4">{error}</p>
                <a
                    href="/problems"
                    className="inline-block px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                >
                    Back to Problems
                </a>
            </div>
        );
    }

    return (
        <div className="max-w-md mx-auto mt-20 p-6 text-center">
            <Shuffle size={48} className="mx-auto mb-4 animate-spin" />
            <h1 className="text-2xl font-bold mb-2">Finding Random Problem</h1>
            <p className="text-muted-foreground mb-4">Loading a random problem for you...</p>
            <Skeleton className="h-4 w-48 mx-auto" />
        </div>
    );
}
