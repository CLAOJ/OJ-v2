/**
 * Base JsonLd Component
 * Injects JSON-LD structured data into the page
 */

interface JsonLdObj {
  '@context': string;
  '@type': string;
  [key: string]: unknown;
}

interface JsonLdProps {
  data: JsonLdObj;
}

export default function JsonLd({ data }: JsonLdProps) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}
