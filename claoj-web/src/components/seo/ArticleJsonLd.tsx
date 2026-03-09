/**
 * Article JSON-LD Component
 * Adds structured data for blog post pages using BlogPosting schema
 */

import JsonLd from './JsonLd';
import type { BlogPostDetail } from '@/types';

interface ArticleJsonLdProps {
  article: BlogPostDetail;
}

export default function ArticleJsonLd({ article }: ArticleJsonLdProps) {
  const articleData = {
    '@context': 'https://schema.org',
    '@type': 'BlogPosting',
    headline: article.title,
    description: article.summary || article.content.slice(0, 200) + '...',
    datePublished: article.publish_on,
    author: article.authors.map((a) => ({
      '@type': 'Person',
      name: a.username,
      url: `${typeof window !== 'undefined' ? window.location.origin : ''}/user/${a.username}`,
    })),
    publisher: {
      '@type': 'Organization',
      name: 'CLAOJ',
      logo: {
        '@type': 'ImageObject',
        url: 'https://beta.claoj.edu.vn/static/icons/og_img.png',
      },
    },
    url: `${typeof window !== 'undefined' ? window.location.origin : ''}/blog/${article.id}`,
    mainEntityOfPage: {
      '@type': 'WebPage',
      '@id': `${typeof window !== 'undefined' ? window.location.origin : ''}/blog/${article.id}`,
    },
    keywords: 'competitive programming, algorithms, blog, tutorial',
    wordCount: article.content.length,
  };

  return <JsonLd data={articleData} />;
}
