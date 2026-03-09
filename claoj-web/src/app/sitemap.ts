import { MetadataRoute } from 'next';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';
const LOCALES = ['en', 'vi'];

// Static routes that exist for each locale
const STATIC_ROUTES = [
  // Homepage
  { route: '', priority: 1.0, changefreq: 'daily' as const },

  // Problems
  { route: '/problems', priority: 0.9, changefreq: 'daily' as const },

  // Contests
  { route: '/contests', priority: 0.9, changefreq: 'hourly' as const },
  { route: '/contests/calendar', priority: 0.7, changefreq: 'hourly' as const },

  // Users & Ratings
  { route: '/users', priority: 0.8, changefreq: 'daily' as const },
  { route: '/ratings', priority: 0.8, changefreq: 'daily' as const },
  { route: '/organizations', priority: 0.7, changefreq: 'weekly' as const },

  // Submissions
  { route: '/submissions', priority: 0.7, changefreq: 'hourly' as const },

  // Blog
  { route: '/blog', priority: 0.8, changefreq: 'daily' as const },

  // Stats
  { route: '/stats', priority: 0.6, changefreq: 'daily' as const },

  // Auth pages (lower priority, noindex recommended but included for completeness)
  { route: '/login', priority: 0.3, changefreq: 'monthly' as const },
  { route: '/register', priority: 0.3, changefreq: 'monthly' as const },
];

/**
 * Generate sitemap entries for all locales
 */
function generateSitemapEntries(): MetadataRoute.Sitemap {
  const entries: MetadataRoute.Sitemap = [];

  for (const locale of LOCALES) {
    for (const { route, priority, changefreq } of STATIC_ROUTES) {
      entries.push({
        url: `${SITE_URL}/${locale}${route}`,
        lastModified: new Date(),
        changeFrequency: changefreq,
        priority,
      });
    }
  }

  return entries;
}

export default function sitemap(): MetadataRoute.Sitemap {
  return generateSitemapEntries();
}
