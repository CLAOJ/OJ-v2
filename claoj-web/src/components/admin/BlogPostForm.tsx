'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Switch } from '@/components/ui/Switch';
import { MarkdownImageButton } from '@/components/admin/MarkdownImageButton';

export interface BlogPostFormData {
    title: string;
    slug: string;
    content: string;
    summary: string;
    /** `datetime-local` value; converted to ISO on submit. */
    publish_on: string;
    visible: boolean;
}

export interface BlogPostFormProps {
    initialData?: Partial<BlogPostFormData>;
    onSubmit: (data: BlogPostFormData) => Promise<void>;
    isLoading?: boolean;
}

/** Blog post fields shared by the authoring screens (v1 BlogPostCreate). */
export default function BlogPostForm({ initialData, onSubmit, isLoading }: BlogPostFormProps) {
    const t = useTranslations('Admin');
    const tCommon = useTranslations('Common');
    const [form, setForm] = useState<BlogPostFormData>({
        title: initialData?.title ?? '',
        slug: initialData?.slug ?? '',
        content: initialData?.content ?? '',
        summary: initialData?.summary ?? '',
        publish_on: initialData?.publish_on ?? '',
        visible: initialData?.visible ?? true,
    });

    const update = <K extends keyof BlogPostFormData>(key: K, value: BlogPostFormData[K]) =>
        setForm(prev => ({ ...prev, [key]: value }));

    const submit = (e: React.FormEvent) => {
        e.preventDefault();
        onSubmit(form);
    };

    return (
        <form onSubmit={submit} className="space-y-6">
            <div className="bg-card rounded-md border p-6 space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-2">
                        <Label htmlFor="blog-title">{t('blogPosts.titleLabel')}</Label>
                        <Input
                            id="blog-title"
                            value={form.title}
                            onChange={e => update('title', e.target.value)}
                            placeholder={t('blogPosts.titlePlaceholder')}
                            required
                        />
                    </div>
                    <div className="grid gap-2">
                        <Label htmlFor="blog-slug">{t('blogPosts.slugLabel')}</Label>
                        <Input
                            id="blog-slug"
                            value={form.slug}
                            onChange={e => update('slug', e.target.value)}
                            placeholder={t('blogPosts.slugPlaceholder')}
                            required
                        />
                    </div>
                </div>

                <div className="grid gap-2">
                    <div className="flex items-center justify-between">
                        <Label htmlFor="blog-summary">{t('blogPosts.summaryLabel')}</Label>
                        <MarkdownImageButton
                            id="blog-summary"
                            value={form.summary}
                            onChange={v => update('summary', v)}
                        />
                    </div>
                    <Textarea
                        id="blog-summary"
                        value={form.summary}
                        onChange={e => update('summary', e.target.value)}
                        rows={3}
                        placeholder={t('blogPosts.summaryPlaceholder')}
                    />
                </div>

                <div className="grid gap-2">
                    <div className="flex items-center justify-between">
                        <Label htmlFor="blog-content">{t('blogPosts.contentLabel')}</Label>
                        <MarkdownImageButton
                            id="blog-content"
                            value={form.content}
                            onChange={v => update('content', v)}
                        />
                    </div>
                    <Textarea
                        id="blog-content"
                        value={form.content}
                        onChange={e => update('content', e.target.value)}
                        rows={14}
                        className="font-mono text-sm"
                        placeholder={t('blogPosts.contentPlaceholder')}
                        required
                    />
                </div>

                <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-2">
                        <Label htmlFor="blog-publish-on">{t('blogPosts.publishDateLabel')}</Label>
                        <Input
                            id="blog-publish-on"
                            type="datetime-local"
                            value={form.publish_on}
                            onChange={e => update('publish_on', e.target.value)}
                        />
                    </div>
                    <label className="flex items-center gap-2 self-end pb-2 cursor-pointer">
                        <Switch
                            checked={form.visible}
                            onCheckedChange={checked => update('visible', checked)}
                        />
                        <span className="text-sm font-medium">{t('blogPosts.visibleLabel')}</span>
                    </label>
                </div>
            </div>

            <div className="flex justify-end">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? t('contestForm.savingButton') : tCommon('create')}
                </button>
            </div>
        </form>
    );
}