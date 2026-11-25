import type { NextConfig } from "next";
import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const nextConfig: NextConfig = {
  reactCompiler: false,
  async redirects() {
    // The plagiarism page used to live at /admin/moss; keep old bookmarks working.
    return [
      { source: '/admin/moss', destination: '/admin/plagiarism', permanent: true },
      { source: '/:locale(en|vi)/admin/moss', destination: '/:locale/admin/plagiarism', permanent: true },
    ];
  },
};

export default withNextIntl(nextConfig);
