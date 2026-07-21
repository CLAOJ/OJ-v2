import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata } from 'next';
import BlogPageContent from './BlogPageContent';
import { ArticleJsonLd } from '@/components/seo';
import api from '@/lib/api';
import { parseLeadingId } from '@/utils/route';

const SITE_URL = process.env.SITE_URL || 'https://beta.claoj.edu.vn';

// Fetch blog post data for metadata (server-side)
async function fetchBlogPost(id: string) {
  try {
    const res = await api.get(`/blog/${id}`);
    return res.data;
  } catch (error) {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; slug: string }>;
}): Promise<Metadata> {
  const { locale, slug } = await params;
  const id = parseLeadingId(slug);

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
        url: `${SITE_URL}/post/${slug}`,
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
  params: Promise<{ locale: string; slug: string }>;
}) {
  const { locale } = await params;
  const messages = await getMessages({ locale });
  const { slug } = await params;
  const id = parseLeadingId(slug);

  // Fetch blog post data for JSON-LD
  const post = await fetchBlogPost(id);

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {post && <ArticleJsonLd article={post} />}
      <BlogPageContent params={params} />
    </NextIntlClientProvider>
  );
}
