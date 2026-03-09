/**
 * Contest JSON-LD Component
 * Adds structured data for contest pages using Event schema
 */

import JsonLd from './JsonLd';
import type { ContestDetail } from '@/types';

interface ContestJsonLdProps {
  contest: ContestDetail;
}

export default function ContestJsonLd({ contest }: ContestJsonLdProps) {
  const contestData = {
    '@context': 'https://schema.org',
    '@type': 'Event',
    name: contest.name,
    description: contest.summary || contest.description || `Participate in ${contest.name} on CLAOJ`,
    startDate: contest.start_time,
    endDate: contest.end_time,
    eventStatus: 'https://schema.org/EventScheduled',
    eventAttendanceMode: 'https://schema.org/OnlineEventAttendanceMode',
    url: `${typeof window !== 'undefined' ? window.location.origin : ''}/contests/${contest.key}`,
    organizer: {
      '@type': 'Organization',
      name: 'CLAOJ',
      url: 'https://beta.claoj.edu.vn',
    },
    performer: {
      '@type': 'Organization',
      name: 'CLAOJ',
    },
    keywords: [
      'competitive programming',
      'contest',
      contest.format || 'ICPC',
      contest.is_rated ? 'rated' : 'unrated',
    ].join(', '),
  };

  return <JsonLd data={contestData} />;
}
