/**
 * Person JSON-LD Component
 * Adds structured data for user profile pages using Person schema
 */

import JsonLd from './JsonLd';
import type { UserDetail } from '@/types';

interface PersonJsonLdProps {
  user: UserDetail;
}

export default function PersonJsonLd({ user }: PersonJsonLdProps) {
  const personData = {
    '@context': 'https://schema.org',
    '@type': 'Person',
    name: user.display_name || user.username,
    alternateName: user.username,
    url: `${typeof window !== 'undefined' ? window.location.origin : ''}/user/${user.username}`,
    description: user.about || `Profile of competitive programmer ${user.username} on CLAOJ`,
    memberSince: user.date_joined,
    award: user.rating ? `${user.rating} rating` : undefined,
    knowsAbout: ['competitive programming', 'algorithms', 'problem solving', 'computer science'],
    image: {
      '@type': 'ImageObject',
      url: `https://www.gravatar.com/avatar/${user.email_hash}?s=400&d=identicon`,
    },
  };

  return <JsonLd data={personData} />;
}
