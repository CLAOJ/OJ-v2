'use client';

import Editor from '@monaco-editor/react';
import { useTheme } from 'next-themes';
import { resolveLanguage } from '@/lib/codeLanguage';

interface CodeEditorProps {
    value: string;
    onChange: (value: string | undefined) => void;
    language?: string;
}

export default function CodeEditor({ value, onChange, language = 'cpp' }: CodeEditorProps) {
    // Use `resolvedTheme`, not `theme`: on first load with defaultTheme="system"
    // `theme` is "system", which would leave the editor on the light Monaco theme
    // over a dark page until the user toggles.
    const { resolvedTheme } = useTheme();
    const resolved = resolveLanguage(language);

    return (
        <div className="h-full border rounded-xl overflow-hidden bg-card">
            <Editor
                height="100%"
                // Controlled `language`, not `defaultLanguage` (which only applies
                // on mount); the prop is a DMOJ key like CPP17, so map it first.
                language={resolved === 'text' ? 'plaintext' : resolved}
                theme={resolvedTheme === 'dark' ? 'vs-dark' : 'light'}
                value={value}
                onChange={onChange}
                options={{
                    fontSize: 14,
                    fontFamily: 'var(--font-mono)',
                    minimap: { enabled: false },
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    padding: { top: 16, bottom: 16 },
                }}
            />
        </div>
    );
}
