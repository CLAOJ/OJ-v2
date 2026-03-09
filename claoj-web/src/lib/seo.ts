/**
 * SEO Utilities for CLAOJ
 * Centralized metadata templates, JSON-LD generators, and i18n helpers
 */

// JSON-LD type definition
export interface JsonLdObj {
  '@context': string;
  '@type': string;
  [key: string]: unknown;
}

// Default SEO metadata template
export const DEFAULT_SEO = {
  title: 'CLAOJ - Online Judge',
  description: 'Modern, high-performance competitive programming platform.',
  openGraph: {
    type: 'website' as const,
    locale: 'en_US' as const,
    url: 'https://beta.claoj.edu.vn',
    siteName: 'CLAOJ',
    images: [
      {
        url: '/static/icons/og_img.png',
        width: 1200,
        height: 630,
        alt: 'CLAOJ - Online Judge',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image' as const,
    title: 'CLAOJ - Online Judge',
    description: 'Modern, high-performance competitive programming platform.',
    images: ['/static/icons/og_img.png'],
  },
};

// Site configuration
export const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';
export const SITE_NAME = 'CLAOJ';

// Locale configurations
export const LOCALES = {
  en: { name: 'English', locale: 'en_US' },
  vi: { name: 'Tiếng Việt', locale: 'vi_VN' },
};

/**
 * Generate canonical URL for a page
 */
export function generateCanonicalUrl(path: string, locale: string): string {
  return `${SITE_URL}/${locale}${path}`;
}

/**
 * Generate JSON-LD for a Problem (SoftwareApplication schema)
 */
export function generateProblemJsonLd(problem: {
  code: string;
  name: string;
  description?: string;
  points: number;
  ac_rate?: number;
  authors?: { username: string }[];
}): JsonLdObj {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: `${problem.code}: ${problem.name}`,
    applicationCategory: 'EducationalApplication',
    description: problem.description || `Solve ${problem.code}: ${problem.name} on CLAOJ`,
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
      url: `${SITE_URL}/user/${a.username}`,
    })),
    keywords: ['competitive programming', 'algorithms', problem.code, problem.name],
  };
}

/**
 * Generate JSON-LD for a Contest (Event schema)
 */
export function generateContestJsonLd(contest: {
  name: string;
  description?: string;
  start_time: string;
  end_time: string;
  key: string;
  format?: string;
  is_rated?: boolean;
}): JsonLdObj {
  return {
    '@context': 'https://schema.org',
    '@type': 'Event',
    name: contest.name,
    description: contest.description || `Participate in ${contest.name} on CLAOJ`,
    startDate: contest.start_time,
    endDate: contest.end_time,
    eventStatus: 'https://schema.org/EventScheduled',
    eventAttendanceMode: 'https://schema.org/OnlineEventAttendanceMode',
    url: `${SITE_URL}/contests/${contest.key}`,
    organizer: {
      '@type': 'Organization',
      name: SITE_NAME,
      url: SITE_URL,
    },
    performer: {
      '@type': 'Organization',
      name: SITE_NAME,
    },
    keywords: [
      'competitive programming',
      'contest',
      contest.format || 'ICPC',
      contest.is_rated ? 'rated' : 'unrated',
    ].join(', '),
  };
}

/**
 * Generate JSON-LD for a Person/UserProfile (Person schema)
 */
export function generatePersonJsonLd(user: {
  username: string;
  display_name?: string;
  rating?: number | null;
  rank?: number | null;
  about?: string;
  date_joined?: string;
}): JsonLdObj {
  return {
    '@context': 'https://schema.org',
    '@type': 'Person',
    name: user.display_name || user.username,
    alternateName: user.username,
    url: `${SITE_URL}/user/${user.username}`,
    description: user.about || `Profile of ${user.username} on CLAOJ`,
    memberSince: user.date_joined,
    award: user.rating ? `${user.rating} rating` : undefined,
    knowsAbout: ['competitive programming', 'algorithms', 'problem solving'],
  };
}

/**
 * Generate JSON-LD for a Blog Post (Article schema)
 */
export function generateArticleJsonLd(article: {
  title: string;
  content: string;
  summary?: string;
  publish_on: string;
  authors: { username: string }[];
  id: string | number;
}): JsonLdObj {
  return {
    '@context': 'https://schema.org',
    '@type': 'BlogPosting',
    headline: article.title,
    description: article.summary || article.content.slice(0, 200) + '...',
    datePublished: article.publish_on,
    author: article.authors.map((a) => ({
      '@type': 'Person',
      name: a.username,
      url: `${SITE_URL}/user/${a.username}`,
    })),
    publisher: {
      '@type': 'Organization',
      name: SITE_NAME,
      logo: {
        '@type': 'ImageObject',
        url: `${SITE_URL}/static/icons/og_img.png`,
      },
    },
    url: `${SITE_URL}/blog/${article.id}`,
    mainEntityOfPage: {
      '@type': 'WebPage',
      '@id': `${SITE_URL}/blog/${article.id}`,
    },
    keywords: 'competitive programming, algorithms, blog, tutorial',
  };
}

/**
 * Generate JSON-LD for Organization (Organization schema)
 */
export function generateOrganizationJsonLd(): JsonLdObj {
  return {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: SITE_NAME,
    url: SITE_URL,
    logo: {
      '@type': 'ImageObject',
      url: `${SITE_URL}/static/icons/og_img.png`,
      width: 1200,
      height: 630,
    },
    sameAs: [
      'https://github.com/claoj',
    ],
    contactPoint: {
      '@type': 'ContactPoint',
      contactType: 'customer support',
      email: 'support@claoj.edu.vn',
    },
  };
}

/**
 * Generate JSON-LD for WebSite (Website schema)
 */
export function generateWebSiteJsonLd(): JsonLdObj {
  return {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: SITE_NAME,
    url: SITE_URL,
    description: 'Modern, high-performance competitive programming platform.',
    inLanguage: 'en-US',
    potentialAction: {
      '@type': 'SearchAction',
      target: {
        '@type': 'EntryPoint',
        urlTemplate: `${SITE_URL}/en/problems?q={search_term_string}`,
      },
      'query-input': 'required name=search_term_string',
    },
  };
}
