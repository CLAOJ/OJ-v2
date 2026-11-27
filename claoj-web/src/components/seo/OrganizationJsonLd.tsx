/**
 * Organization JSON-LD Component
 * Adds structured data for the CLAOJ organization using Organization schema
 */

import JsonLd from './JsonLd';

export default function OrganizationJsonLd() {
  const organizationData = {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: 'CLAOJ',
    url: 'https://beta.claoj.edu.vn',
    logo: {
      '@type': 'ImageObject',
      url: 'https://beta.claoj.edu.vn/static/icons/og_img.png',
      width: 1200,
      height: 630,
    },
    sameAs: [
      'https://github.com/claoj',
    ],
    description: 'Hệ thống chấm bài tự động CLAOJ, nơi học sinh rèn luyện Tin học cho các kỳ thi HSG, Tin học trẻ, Olympic 30/4.',
  };

  return <JsonLd data={organizationData} />;
}
