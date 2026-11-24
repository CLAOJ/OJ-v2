'use client';

import { useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import axios from 'axios';
import { toast } from 'sonner';
import { ImagePlus } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { adminMediaApi } from '@/lib/adminApi';
import { insertImageMarkdown } from '@/lib/markdownImage';

const MAX_BYTES = 2 * 1024 * 1024;
const ALLOWED = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

interface MarkdownImageButtonProps {
    /** id of the textarea the markdown is inserted into. */
    id: string;
    value: string;
    onChange: (value: string) => void;
}

/** Uploads an image and inserts `![alt](url)` at the textarea cursor. */
export function MarkdownImageButton({ id, value, onChange }: MarkdownImageButtonProps) {
    const t = useTranslations('Admin');
    const inputRef = useRef<HTMLInputElement>(null);
    const valueRef = useRef(value);
    valueRef.current = value;
    const [uploading, setUploading] = useState(false);

    const handleFile = async (file: File | undefined) => {
        if (!file) return;
        if (!ALLOWED.includes(file.type)) {
            toast.error(t('markdownImage.invalidType'));
            return;
        }
        if (file.size > MAX_BYTES) {
            toast.error(t('markdownImage.tooLarge'));
            return;
        }

        setUploading(true);
        try {
            const response = await adminMediaApi.uploadImage(file);
            const current = valueRef.current;
            const textarea = document.getElementById(id) as HTMLTextAreaElement | null;
            const start = textarea?.selectionStart ?? current.length;
            const end = textarea?.selectionEnd ?? current.length;
            const next = insertImageMarkdown(
                current,
                start,
                end,
                file.name.replace(/\.[^.]+$/, ''),
                response.data.url,
            );
            onChange(next.value);
            requestAnimationFrame(() => {
                const el = document.getElementById(id) as HTMLTextAreaElement | null;
                el?.focus();
                el?.setSelectionRange(next.cursor, next.cursor);
            });
        } catch (error) {
            const serverError = axios.isAxiosError<{ error?: string }>(error) ? error.response?.data?.error : undefined;
            toast.error(serverError || t('markdownImage.uploadFailed'));
        } finally {
            setUploading(false);
        }
    };

    return (
        <>
            <input
                ref={inputRef}
                type="file"
                accept="image/png,image/jpeg,image/gif,image/webp"
                className="hidden"
                onChange={(e) => {
                    const file = e.target.files?.[0];
                    e.target.value = '';
                    handleFile(file);
                }}
            />
            <Button
                type="button"
                variant="outline"
                size="sm"
                loading={uploading}
                onClick={() => inputRef.current?.click()}
            >
                <ImagePlus className="h-4 w-4 mr-1" />
                {uploading ? t('markdownImage.uploading') : t('markdownImage.button')}
            </Button>
        </>
    );
}