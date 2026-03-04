'use client';

import Editor from '@monaco-editor/react';
import { useTheme } from 'next-themes';

interface CodeEditorProps {
    value: string;
    onChange: (value: string | undefined) => void;
    language?: string;
}

export default function CodeEditor({ value, onChange, language = 'cpp' }: CodeEditorProps) {
    const { theme } = useTheme();

    return (
        <div className="h-full border rounded-xl overflow-hidden bg-card">
            <Editor
                height="100%"
                defaultLanguage={language}
                theme={theme === 'dark' ? 'vs-dark' : 'light'}
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
