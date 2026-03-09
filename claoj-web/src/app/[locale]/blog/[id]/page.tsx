import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import BlogPageContent from './BlogPageContent';
import { ArticleJsonLd } from '@/components/seo';
import api from '@/lib/api';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch blog post data for metadata (server-side)
async function fetchBlogPost(id: string) {
  try {
    const res = await api.get(`http://localhost:8080/api/v2/blog/${id}`);
    return res.data;
  } catch (error) {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}): Promise<Metadata> {
  const { locale, id } = await params;

  // Try to fetch blog post data
  const post = await fetchBlogPost(id);

  if (post) {
    const title = `${post.title} | CLAOJ Blog`;
    const description = post.summary || post.content.slice(0, 200) + '...';
    const authorNames = post.authors?.map((a: { username: string }) => a.username).join(', ') || 'CLAOJ';

    return {
      title,
      description,
      openGraph: {
        title,
        description,
        type: 'article',
        publishedTime: post.publish_on,
        authors: authorNames ? [`@${authorNames}`] : [],
        locale: locale === 'vi' ? 'vi_VN' : 'en_US',
        url: `${SITE_URL}/${locale}/blog/${id}`,
        siteName: 'CLAOJ',
      },
    };
  }

  return {
    title: `Blog Post | CLAOJ`,
    description: 'Competitive programming blog post on CLAOJ',
  };
}

export default async function BlogPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}) {
  const { locale } = await params;
  const messages = await getMessages({ locale });
  const { id } = await params;

  // Fetch blog post data for JSON-LD
  const post = await fetchBlogPost(id);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {post && <ArticleJsonLd article={post} />}
      <BlogPageContent params={params} />
    </NextIntlClientProvider>
  );
}
