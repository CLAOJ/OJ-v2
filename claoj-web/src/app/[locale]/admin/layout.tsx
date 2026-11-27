import ScopedIntl from '@/components/providers/ScopedIntl';
import AdminShell from './AdminShell';

// Server wrapper: ships the Admin/admin translations to the admin section only.
export default async function AdminLayout({
    children,
    params,
}: {
    children: React.ReactNode;
    params: Promise<{ locale: string }>;
}) {
    const { locale } = await params;
    return (
        <ScopedIntl locale={locale} include={['Admin', 'admin']}>
            <AdminShell>{children}</AdminShell>
        </ScopedIntl>
    );
}
