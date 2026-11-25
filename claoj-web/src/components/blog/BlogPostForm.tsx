'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { blogAuthoringApi } from '@/lib/api';
import { useAuth } from '@/components/providers/AuthProvider';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Select } from '@/components/ui/Select';
import { Switch } from '@/components/ui/Switch';
import { Textarea } from '@/components/ui/Textarea';
import { useRouter } from '@/navigation';

/** v1 settings.CLAOJ_BLOG_MIN_PROBLEM_COUNT; the API enforces it too. */
export const BLOG_MIN_PROBLEM_COUNT = 10;

export interface BlogPostFormValues {
    title: string;
    content: string;
    publish_on: string;
    visible: boolean;
    organization_id: number | null;
    organization_name?: string;
}

interface BlogPostFormProps {
    mode: 'create' | 'edit';
    postId?: number;
    initial?: BlogPostFormValues;
}

function toLocalInput(iso: string | undefined): string {
    const date = iso ? new Date(iso) : new Date();
    const offset = date.getTimezoneOffset() * 60_000;
    return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

/** v1 blog/edit.html: title, publish_on, visible, content (+ organization). */
export default function BlogPostForm({ mode, postId, initial }: BlogPostFormProps) {
    const t = useTranslations('Blog');
    const router = useRouter();
    const { user } = useAuth();
    const [values, setValues] = useState<BlogPostFormValues>(initial ?? {
        title: '',
        content: '',
        publish_on: new Date().toISOString(),
        visible: false,
        organization_id: null,
    });
    const [saving, setSaving] = useState(false);

    const adminOf = user?.admin_of ?? [];
    // Keep the post's current organization selectable for an author who is not
    // its admin (the API accepts an unchanged value).
    const orgOptions = values.organization_id && !adminOf.some(o => o.id === values.organization_id)
        ? [...adminOf, { id: values.organization_id, name: values.organization_name ?? `#${values.organization_id}`, slug: '' }]
        : adminOf;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!values.title.trim() || !values.content.trim()) return;
        setSaving(true);
        try {
            const payload = {
                title: values.title,
                content: values.content,
                publish_on: new Date(values.publish_on).toISOString(),
                visible: values.visible,
                organization_id: values.organization_id,
            };
            const res = mode === 'create'
                ? await blogAuthoringApi.create(payload)
                : await blogAuthoringApi.update(postId as number, payload);
            toast.success(mode === 'create' ? t('created') : t('updated'));
            router.push(`/post/${res.data.blog_post.id}-${res.data.blog_post.slug}`);
        } catch (error) {
            const message = (error as { response?: { data?: { error?: string } } })?.response?.data?.error;
            toast.error(message || t('saveFailed'));
            setSaving(false);
        }
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid gap-2">
                <Label htmlFor="title">{t('postTitle')}</Label>
                <Input
                    id="title"
                    value={values.title}
                    onChange={(e) => setValues({ ...values, title: e.target.value })}
                    placeholder={t('postTitlePlaceholder')}
                    required
                />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="grid gap-2">
                    <Label htmlFor="publish_on">{t('publishOn')}</Label>
                    <Input
                        id="publish_on"
                        type="datetime-local"
                        value={toLocalInput(values.publish_on)}
                        onChange={(e) => setValues({ ...values, publish_on: e.target.value })}
                    />
                </div>
                <div className="grid gap-2">
                    <Label htmlFor="organization">{t('organization')}</Label>
                    <Select
                        id="organization"
                        value={values.organization_id ?? ''}
                        disabled={orgOptions.length === 0}
                        onChange={(e) => {
                            const id = e.target.value ? Number(e.target.value) : null;
                            setValues({
                                ...values,
                                organization_id: id,
                                organization_name: orgOptions.find(o => o.id === id)?.name,
                            });
                        }}
                    >
                        <option value="">{t('noOrganization')}</option>
                        {orgOptions.map(org => (
                            <option key={org.id} value={org.id}>{org.name}</option>
                        ))}
                    </Select>
                    {orgOptions.length === 0 && (
                        <p className="text-xs text-muted-foreground">{t('noOrganizationHint')}</p>
                    )}
                </div>
            </div>

            <div className="flex items-center gap-2">
                <Switch
                    id="visible"
                    checked={values.visible}
                    onCheckedChange={(checked) => setValues({ ...values, visible: checked })}
                />
                <Label htmlFor="visible">{t('visible')}</Label>
            </div>

            <div className="grid gap-2">
                <Label htmlFor="content">{t('content')}</Label>
                <Textarea
                    id="content"
                    value={values.content}
                    onChange={(e) => setValues({ ...values, content: e.target.value })}
                    rows={18}
                    className="font-mono text-sm"
                    placeholder={t('contentPlaceholder')}
                    required
                />
            </div>

            <div className="flex justify-end gap-3">
                <Button type="button" variant="outline" onClick={() => router.push('/post')}>
                    {t('cancel')}
                </Button>
                <Button type="submit" disabled={saving || !values.title.trim() || !values.content.trim()}>
                    {saving ? t('saving') : mode === 'create' ? t('create') : t('save')}
                </Button>
            </div>
        </form>
    );
}