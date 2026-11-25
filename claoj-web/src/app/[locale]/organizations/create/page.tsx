'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { organizationsApi } from '@/lib/api';
import { useAuth } from '@/components/providers/AuthProvider';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Link, useRouter } from '@/navigation';
import { Building2, Lock } from 'lucide-react';
import { Skeleton } from '@/components/ui/Skeleton';

/** v1 `/organizations/create` (CreateOrganization): judge.add_organization. */
export default function CreateOrganizationPage() {
    const t = useTranslations('Organizations');
    const router = useRouter();
    const { user, loading } = useAuth();
    const [saving, setSaving] = useState(false);
    const [form, setForm] = useState({
        name: '',
        slug: '',
        about: '',
        logo_override_image: '',
    });

    if (loading) return <div className="max-w-2xl mx-auto p-8"><Skeleton className="h-96 w-full rounded-md" /></div>;

    const allowed = !!user && (user.is_superuser === true || (user.permissions ?? []).includes('judge.add_organization'));
    if (!allowed) {
        return (
            <div className="max-w-2xl mx-auto py-20 text-center space-y-3">
                <Lock size={48} className="mx-auto text-muted-foreground opacity-30" />
                <p className="font-medium">{t('createNotAllowed')}</p>
                <Link href="/organizations" className="text-primary font-medium hover:underline">
                    {t('backToOrganizations')}
                </Link>
            </div>
        );
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!form.name.trim() || !form.slug.trim()) return;
        setSaving(true);
        try {
            const res = await organizationsApi.create({
                name: form.name,
                slug: form.slug,
                about: form.about,
                logo_override_image: form.logo_override_image,
            });
            toast.success(t('created'));
            router.push(`/organization/${res.data.organization.id}-${res.data.organization.slug}`);
        } catch (error) {
            const message = (error as { response?: { data?: { error?: string } } })?.response?.data?.error;
            toast.error(message || t('createFailed'));
            setSaving(false);
        }
    };

    return (
        <div className="max-w-2xl mx-auto space-y-8 mt-4 pb-20">
            <header className="space-y-2">
                <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                    <Building2 className="text-primary" size={40} />
                    {t('createTitle')}
                </h1>
                <p className="text-muted-foreground font-medium opacity-80">{t('createSubtitle')}</p>
            </header>

            <form onSubmit={handleSubmit} className="bg-card border rounded-md p-6 shadow-sm space-y-6">
                <div className="grid gap-2">
                    <Label htmlFor="name">{t('nameLabel')}</Label>
                    <Input
                        id="name"
                        value={form.name}
                        onChange={(e) => setForm({ ...form, name: e.target.value })}
                        required
                    />
                </div>

                <div className="grid gap-2">
                    <Label htmlFor="slug">{t('slugLabel')}</Label>
                    <Input
                        id="slug"
                        value={form.slug}
                        onChange={(e) => setForm({ ...form, slug: e.target.value })}
                        placeholder={t('slugPlaceholder')}
                        required
                    />
                </div>

                <div className="grid gap-2">
                    <Label htmlFor="about">{t('aboutLabel')}</Label>
                    <Textarea
                        id="about"
                        value={form.about}
                        onChange={(e) => setForm({ ...form, about: e.target.value })}
                        rows={6}
                        className="text-sm"
                    />
                </div>

                <div className="grid gap-2">
                    <Label htmlFor="logo">{t('logoLabel')}</Label>
                    <Input
                        id="logo"
                        value={form.logo_override_image}
                        onChange={(e) => setForm({ ...form, logo_override_image: e.target.value })}
                        placeholder={t('logoPlaceholder')}
                    />
                </div>

                <div className="flex justify-end gap-3">
                    <Button type="button" variant="outline" onClick={() => router.push('/organizations')}>
                        {t('cancel')}
                    </Button>
                    <Button type="submit" disabled={saving || !form.name.trim() || !form.slug.trim()}>
                        {saving ? t('creating') : t('create')}
                    </Button>
                </div>
            </form>
        </div>
    );
}