/**
 * Problem JSON-LD Component
 * Adds structured data for problem pages using SoftwareApplication schema
 */

import JsonLd from './JsonLd';
import type { ProblemDetail } from '@/types';

interface ProblemJsonLdProps {
  problem: ProblemDetail;
}

export default function ProblemJsonLd({ problem }: ProblemJsonLdProps) {
  const problemData = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: `${problem.code}: ${problem.name}`,
    applicationCategory: 'EducationalApplication',
    description: `Solve ${problem.code}: ${problem.name} - A competitive programming problem on CLAOJ`,
    offers: {
      '@type': 'Offer',
      price: '0',
      priceCurrency: 'USD',
    },
    aggregateRating: {
      '@type': 'AggregateRating',
      ratingValue: problem.ac_rate ? Math.round(problem.ac_rate) : 0,
      bestRating: 100,
      worstRating: 0,
    },
    author: problem.authors?.map((a) => ({
      '@type': 'Person',
      name: a.username,
      url: `${typeof window !== 'undefined' ? window.location.origin : ''}/user/${a.username}`,
    })),
    keywords: ['competitive programming', 'algorithms', problem.code, problem.name].join(', '),
    teaches: ['Algorithm Design', 'Problem Solving', 'Computer Science'],
  };

  return <JsonLd data={problemData} />;
}
