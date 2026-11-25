'use client';

import { useParams } from 'next/navigation';
import TicketListContent from '@/components/tickets/TicketListContent';

/** v1 /problem/<p>/tickets: the tickets linked to one problem. */
export default function ProblemTicketsPage() {
    const params = useParams();
    return <TicketListContent problem={params.code as string} />;
}