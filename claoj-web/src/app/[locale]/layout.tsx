import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import type { Metadata, Viewport } from "next";
import { Outfit } from "next/font/google";
import "./globals.css";
import QueryProvider from "@/components/providers/QueryProvider";
import { AuthProvider } from "@/components/providers/AuthProvider";
import { ThemeProvider } from "@/components/providers/ThemeProvider";
import { WebSocketProvider } from "@/contexts/WebSocketContext";
import Navbar from "@/components/layout/Navbar";
import Footer from "@/components/layout/Footer";

const outfit = Outfit({
  subsets: ["latin", "latin-ext"],
  variable: "--font-outfit",
});

const SITE_URL = process.env.SITE_URL || "https://beta.claoj.edu.vn";

export const metadata: Metadata = {
  title: "CLAOJ - Online Judge",
  description: "Modern, high-performance competitive programming platform.",
  metadataBase: new URL(SITE_URL),

  // Open Graph
  openGraph: {
    title: "CLAOJ - Online Judge",
    description: "Modern, high-performance competitive programming platform.",
    siteName: "CLAOJ",
    images: [
      {
        url: "/static/icons/og_img.png",
        width: 1200,
        height: 630,
        alt: "CLAOJ",
      },
    ],
    locale: "en_US",
    type: "website",
  },

  // Twitter Card
  twitter: {
    card: "summary_large_image",
    title: "CLAOJ - Online Judge",
    description: "Modern, high-performance competitive programming platform.",
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
  const messages = await getMessages({ locale });

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className={`${outfit.variable} font-sans antialiased min-h-screen flex flex-col`}>
        <NextIntlClientProvider locale={locale} messages={messages}>
          <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
            <QueryProvider>
              <AuthProvider>
                <WebSocketProvider>
                  <Navbar />
                  <main className="flex-grow container mx-auto px-4 py-8">
                    {children}
                  </main>
                  <Footer />
                </WebSocketProvider>
              </AuthProvider>
            </QueryProvider>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
