import { MetadataRoute } from 'next';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: '*',
        allow: ['/', '/problems', '/contests', '/users', '/ratings', '/blog', '/stats', '/organizations', '/submissions'],
        disallow: ['/settings', '/register', '/admin', '/api', '/ticket', '/tickets', '/notifications'],
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
  };
}
