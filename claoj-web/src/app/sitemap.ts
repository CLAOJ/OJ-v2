import { MetadataRoute } from 'next';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Static routes (single prefix-less URL per page, like v1)
const STATIC_ROUTES = [
  { route: '', priority: 1.0, changefreq: 'daily' as const },
  { route: '/problems', priority: 0.9, changefreq: 'daily' as const },
  { route: '/contests', priority: 0.9, changefreq: 'hourly' as const },
  { route: '/contests/calendar', priority: 0.7, changefreq: 'hourly' as const },
  { route: '/users', priority: 0.8, changefreq: 'daily' as const },
  { route: '/ratings', priority: 0.8, changefreq: 'daily' as const },
  { route: '/organizations', priority: 0.7, changefreq: 'weekly' as const },
  { route: '/submissions', priority: 0.7, changefreq: 'hourly' as const },
  { route: '/post', priority: 0.8, changefreq: 'daily' as const },
  { route: '/stats', priority: 0.6, changefreq: 'daily' as const },
  { route: '/login', priority: 0.3, changefreq: 'monthly' as const },
  { route: '/register', priority: 0.3, changefreq: 'monthly' as const },
];

function generateSitemapEntries(): MetadataRoute.Sitemap {
  return STATIC_ROUTES.map(({ route, priority, changefreq }) => ({
    url: `${SITE_URL}${route}`,
    lastModified: new Date(),
    changeFrequency: changefreq,
    priority,
  }));
}

export default function sitemap(): MetadataRoute.Sitemap {
  return generateSitemapEntries();
}
