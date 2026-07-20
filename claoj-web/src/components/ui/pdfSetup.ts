// Side-effect module: configures pdf.js for react-pdf.
// Isolated from the viewer so Jest can mock it — the `import.meta.url` worker
// reference and the CSS imports below cannot be transformed by ts-jest (CommonJS).
import { pdfjs } from 'react-pdf';
import 'react-pdf/dist/Page/TextLayer.css';
import 'react-pdf/dist/Page/AnnotationLayer.css';

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
    'pdfjs-dist/build/pdf.worker.min.mjs',
    import.meta.url,
).toString();
