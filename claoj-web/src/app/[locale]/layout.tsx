import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getTranslations } from 'next-intl/server';
import { messagesFor } from '@/i18n/scope';
import type { Metadata, Viewport } from "next";
import { Be_Vietnam_Pro } from "next/font/google";
import "./globals.css";
import QueryProvider from "@/components/providers/QueryProvider";
import { AuthProvider } from "@/components/providers/AuthProvider";
import { ThemeProvider } from "@/components/providers/ThemeProvider";
import { WebSocketProvider } from "@/contexts/WebSocketContext";
import { ConfirmProvider } from "@/components/providers/ConfirmProvider";
import { DayjsLocale } from "@/components/providers/DayjsLocale";
import Navbar from "@/components/layout/Navbar";
import { NameStyles } from "@/components/providers/NameStyles";
import { ImpersonationBanner } from "@/components/admin/ImpersonationBanner";
import Footer from "@/components/layout/Footer";
import { DEFAULT_SEO, SITE_URL, generateWebSiteJsonLd, generateOrganizationJsonLd } from '@/lib/seo';
import JsonLd from '@/components/seo/JsonLd';
import { Toaster } from 'sonner';
import { cookies } from 'next/headers';
import api from '@/lib/api';
import type { NavigationItem } from '@/lib/navbar';

// Navbar rows are public, site-wide and rarely edited (the API itself says
// max-age=60), so each worker refetches at most once a minute. Rendering them on
// the server stops the header swapping from the static fallback after load.
let navCache: { at: number; data: NavigationItem[] } | null = null;
async function getNavigation(): Promise<NavigationItem[] | null> {
  if (navCache && Date.now() - navCache.at < 60_000) return navCache.data;
  try {
    const res = await api.get<{ data: NavigationItem[] }>('/navigation', { timeout: 2000 });
    navCache = { at: Date.now(), data: res.data?.data ?? [] };
    return navCache.data;
  } catch {
    return navCache?.data ?? null; // stale beats the fallback; null = fallback
  }
}

const beVietnamPro = Be_Vietnam_Pro({
  subsets: ["latin", "vietnamese"],
  weight: ["300", "400", "500", "600", "700"],
  variable: "--font-be-vietnam-pro",
});

const siteUrl = process.env.SITE_URL || "https://beta.claoj.edu.vn";

export const metadata: Metadata = {
  title: {
    default: "CLAOJ: Long An HSGS Online Judge",
    template: "%s | CLAOJ",
  },
  description: "Web chấm bài tự động trường THPT chuyên Long An, nơi học sinh rèn luyện Tin học cho các kỳ thi HSG, Tin học trẻ, Olympic 30/4.",
  metadataBase: new URL(siteUrl),
  // NOTE: no blanket `alternates.canonical` here — a site-wide root canonical
  // makes every page look like a duplicate of the homepage. Each page sets its
  // own self-canonical in generateMetadata instead.

  // Open Graph
  openGraph: {
    title: "CLAOJ: Long An HSGS Online Judge",
    description: "Web chấm bài tự động trường THPT chuyên Long An, nơi học sinh rèn luyện Tin học cho các kỳ thi HSG, Tin học trẻ, Olympic 30/4.",
    siteName: "CLAOJ",
    images: [
      {
        url: "/static/icons/og_img.png",
        width: 1200,
        height: 630,
        alt: "CLAOJ",
      },
    ],
    locale: "vi_VN",
    type: "website",
  },

  // Twitter Card
  twitter: {
    card: "summary_large_image",
    title: "CLAOJ: Long An HSGS Online Judge",
    description: "Web chấm bài tự động trường THPT chuyên Long An, nơi học sinh rèn luyện Tin học cho các kỳ thi HSG, Tin học trẻ, Olympic 30/4.",
    images: ["/static/icons/og_img.png"],
  },

  // Icons
  icons: {
    icon: [
      { url: "/static/icons/favicon.ico", sizes: "any" },
      { url: "/static/icons/favicon-16x16.png", sizes: "16x16", type: "image/png" },
      { url: "/static/icons/favicon-32x32.png", sizes: "32x32", type: "image/png" },
      { url: "/static/icons/favicon-96x96.png", sizes: "96x96", type: "image/png" },
      { url: "/static/icons/android-chrome-192x192.png", sizes: "192x192", type: "image/png" },
    ],
    apple: [
      { url: "/static/icons/apple-icon.png", sizes: "180x180", type: "image/png" },
      { url: "/static/icons/apple-icon-precomposed.png", sizes: "180x180", type: "image/png" },
    ],
  },

  // Manifest
  manifest: "/static/icons/manifest.json",

  // Other
  appleWebApp: {
    capable: true,
    statusBarStyle: "default",
    title: "CLAOJ",
  },

  // Additional metadata
  other: {
    "msapplication-TileColor": "#263238",
    "msapplication-TileImage": "/static/icons/ms-icon-144x144.png",
    "msapplication-config": "/static/icons/browserconfig.xml",
  },
};

export const viewport: Viewport = {
  themeColor: "#263238",
  width: "device-width",
  initialScale: 1,
};

export default async function RootLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const messages = await messagesFor(locale);
  const tCommon = await getTranslations({ locale, namespace: 'Common' });
  const navigation = await getNavigation();
  // Without a session cookie the visitor is certainly logged out, so the navbar
  // can show Log in / Sign up in the HTML instead of after the /user/me probe.
  const cookieStore = await cookies();
  const hasSession = cookieStore.has('access_token') || cookieStore.has('refresh_token');

  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <JsonLd data={generateWebSiteJsonLd()} />
        <JsonLd data={generateOrganizationJsonLd()} />
      </head>
      <body className={`${beVietnamPro.variable} font-sans antialiased min-h-screen flex flex-col`}>
        <NextIntlClientProvider locale={locale} messages={messages}>
          <DayjsLocale locale={locale} />
          <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
            <QueryProvider>
              <AuthProvider>
                <WebSocketProvider>
                 <ConfirmProvider>
                  <a
                    href="#main-content"
                    className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-primary-foreground"
                  >
                    {tCommon('skipToContent')}
                  </a>
                  <NameStyles />
                  <Navbar initialNavigation={navigation} hasSession={hasSession} />
                  <ImpersonationBanner />
                  <main id="main-content" tabIndex={-1} className="flex-grow container mx-auto px-4 py-8 focus:outline-none">
                    {children}
                  </main>
                  <Footer />
                  {/* Ten components call sonner's toast() for success and
                      error feedback, but the Toaster was never mounted, so
                      every one of those notifications rendered nowhere —
                      failed actions looked like nothing had happened. */}
                  <Toaster position="top-right" richColors closeButton />
                 </ConfirmProvider>
                </WebSocketProvider>
              </AuthProvider>
            </QueryProvider>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
