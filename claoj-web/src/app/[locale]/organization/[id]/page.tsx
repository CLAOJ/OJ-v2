'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import {
    OrganizationDetail,
    OrganizationMember,
    OrganizationProblemRow,
    OrganizationContestRow,
    PaginatedList,
    Submission,
} from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import { SortableHeader } from '@/components/ui/SortableHeader';
import { toggleSort } from '@/lib/listSort';
import {
    DEFAULT_MEMBER_SORT,
    DEFAULT_PROBLEM_SORT,
    formatPoints,
    parseOrgTab,
    type OrgMemberSort,
    type OrgProblemSort,
    type OrgTab,
} from '@/lib/organizationView';
import { Link, useRouter, usePathname } from '@/navigation';
import { useState, use, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import { isAxiosError } from 'axios';
import { parseLeadingId } from '@/utils/route';
import {
    Building2,
    Users,
    UserPlus,
    UserMinus,
    UserCheck,
    Shield,
    Crown,
    TrendingUp,
    Hash,
    ArrowLeft,
    FileText,
    Puzzle,
    Trophy,
    List,
    Lock,
    Settings,
    FilePlus,
    PenLine,
} from 'lucide-react';
import { cn, getRankColor, getStatusVariant } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { motion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { toast } from 'sonner';

dayjs.extend(relativeTime);

const TAB_PAGE_SIZE = PAGE_SIZE_OPTIONS[1];

/** Empty-list placeholder shared by every organization tab. */
function EmptyState({ icon, text }: { icon: React.ReactNode; text: string }) {
    return (
        <div className="text-center py-20 text-muted-foreground border-2 border-dashed rounded-md">
            <div className="mx-auto mb-4 opacity-10 flex justify-center">{icon}</div>
            <p className="font-medium">{text}</p>
        </div>
    );
}

/** Shown when one of the member-only tabs answers 403. */
function MembersOnly() {
    const t = useTranslations('Organization');
    return (
        <div className="text-center py-20 text-muted-foreground border-2 border-dashed rounded-md">
            <Lock size={48} className="mx-auto mb-4 opacity-10" />
            <p className="font-medium">{t('membersOnly')}</p>
            <p className="text-sm opacity-70 mt-1">{t('membersOnlyHint')}</p>
        </div>
    );
}

interface MemberTableProps {
    members: OrganizationMember[];
    /** The member tab sorts server-side; the admins tab keeps its own order. */
    sort?: { sort: OrgMemberSort; order: 'asc' | 'desc' };
    onSort?: (column: OrgMemberSort) => void;
}

function MemberTable({ members, sort, onSort }: MemberTableProps) {
    const t = useTranslations('Organization');
    const sortable = !!sort && !!onSort;
    const sortProps = (column: OrgMemberSort) => ({
        active: sort?.sort === column,
        order: sort?.order ?? 'desc',
        onClick: () => onSort?.(column),
    });

    return (
        <div className="bg-card border rounded-md overflow-hidden">
            <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                    <thead>
                        <tr className="bg-muted/30 border-b">
                            <th className="px-6 py-4 text-xs font-medium text-muted-foreground">#</th>
                            <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('member')}</th>
                            {sortable ? (
                                <>
                                    <SortableHeader label={t('rating')} className="text-center w-32" {...sortProps('rating')} />
                                    <SortableHeader label={t('solved')} className="text-center w-32" {...sortProps('problem_count')} />
                                    <SortableHeader label={t('points')} className="text-center w-32" {...sortProps('points')} />
                                    <SortableHeader label={t('pp')} className="text-center w-32" {...sortProps('performance_points')} />
                                </>
                            ) : (
                                <>
                                    <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('rating')}</th>
                                    <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('solved')}</th>
                                    <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('points')}</th>
                                    <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('pp')}</th>
                                </>
                            )}
                            <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-right w-32">{t('role')}</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-muted/50">
                        {members.map((member, index) => (
                            <tr key={member.id} className="hover:bg-muted/10 transition-colors">
                                <td className="px-6 py-4 text-sm font-mono text-muted-foreground">{index + 1}</td>
                                <td className="px-6 py-4">
                                    <Link href={`/user/${member.username}`} className="flex items-center gap-3 group outline-none">
                                        <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center text-primary font-medium text-xs">
                                            {(member.display_name || member.username)[0]?.toUpperCase()}
                                        </div>
                                        <div className="flex flex-col">
                                            <span className="font-medium text-sm group-hover:text-primary transition-colors">
                                                {member.display_name || member.username}
                                            </span>
                                            <span className="text-xs font-mono text-muted-foreground opacity-60">@{member.username}</span>
                                        </div>
                                    </Link>
                                </td>
                                <td className="px-6 py-4 text-center">
                                    <span className={cn('text-sm font-medium', getRankColor(member.rating))}>
                                        {member.rating ?? '-'}
                                    </span>
                                </td>
                                <td className="px-6 py-4 text-center text-sm font-medium">{member.problem_count}</td>
                                <td className="px-6 py-4 text-center text-sm font-medium">{formatPoints(member.points)}</td>
                                <td className="px-6 py-4 text-center text-sm font-semibold text-primary">{formatPoints(member.performance_points)}</td>
                                <td className="px-6 py-4 text-right">
                                    <Badge
                                        className={cn(
                                            'text-xs font-medium',
                                            member.role === 'admin'
                                                ? 'bg-amber-500/10 text-amber-500 border-amber-500/20'
                                                : 'bg-muted text-muted-foreground border-transparent'
                                        )}
                                    >
                                        {member.role === 'admin' ? <Crown size={12} className="inline mr-1" /> : null}
                                        {member.role === 'admin' ? t('admin') : t('member')}
                                    </Badge>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

export default function OrganizationDetailPage({ params }: { params: Promise<{ id: string }> }) {
    const { id: idParam } = use(params);
    const id = parseLeadingId(idParam);
    const t = useTranslations('Organization');
    const tCommon = useTranslations('Common');
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
    const { user } = useAuth();
    // Deep-linkable tabs: ?tab=problems etc. Read once on mount, then the URL
    // follows every tab change (same pattern as the contest page).
    const [activeTab, setActiveTab] = useState<OrgTab>(() => parseOrgTab(searchParams.get('tab')));
    const [memberSort, setMemberSort] = useState<{ sort: OrgMemberSort; order: 'asc' | 'desc' }>(DEFAULT_MEMBER_SORT);
    const [problemSort, setProblemSort] = useState<{ sort: OrgProblemSort; order: 'asc' | 'desc' }>(DEFAULT_PROBLEM_SORT);
    const [tabPage, setTabPage] = useState(1);
    const [tabPageSize, setTabPageSize] = useState(TAB_PAGE_SIZE);
    const [showRequestForm, setShowRequestForm] = useState(false);
    const [reason, setReason] = useState('');
    const queryClient = useQueryClient();

    // Back/forward changes the query string without remounting; follow it.
    useEffect(() => {
        setActiveTab(parseOrgTab(searchParams.get('tab')));
    }, [searchParams]);

    const isAuthenticated = !!user;

    const selectTab = (tab: OrgTab) => {
        setActiveTab(tab);
        setTabPage(1);
        const next = new URLSearchParams(searchParams.toString());
        next.set('tab', tab);
        router.replace(`${pathname}?${next.toString()}`, { scroll: false });
    };

    const { data: org, isLoading: orgLoading } = useQuery({
        queryKey: ['organization', id, memberSort.sort, memberSort.order],
        queryFn: async () => {
            const res = await api.get<OrganizationDetail>(`/organization/${id}`, {
                params: { sort: memberSort.sort, order: memberSort.order },
            });
            return res.data;
        }
    });

    const problemsQuery = useQuery({
        queryKey: ['organization-problems', id, tabPage, tabPageSize, problemSort.sort, problemSort.order],
        queryFn: async () => {
            const res = await api.get<PaginatedList<OrganizationProblemRow>>(`/organization/${id}/problems`, {
                params: { page: tabPage, page_size: tabPageSize, sort: problemSort.sort, order: problemSort.order },
            });
            return res.data;
        },
        enabled: activeTab === 'problems',
    });

    const contestsQuery = useQuery({
        queryKey: ['organization-contests', id, tabPage, tabPageSize],
        queryFn: async () => {
            const res = await api.get<PaginatedList<OrganizationContestRow>>(`/organization/${id}/contests`, {
                params: { page: tabPage, page_size: tabPageSize },
            });
            return res.data;
        },
        enabled: activeTab === 'contests',
    });

    const submissionsQuery = useQuery({
        queryKey: ['organization-submissions', id, tabPage, tabPageSize],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Submission>>(`/organization/${id}/submissions`, {
                params: { page: tabPage, page_size: tabPageSize },
            });
            return res.data;
        },
        enabled: activeTab === 'submissions',
    });

    const joinMutation = useMutation({
        mutationFn: async () => {
            const res = await api.post(`/organization/${id}/join`);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['organization', id] });
        }
    });

    const leaveMutation = useMutation({
        mutationFn: async () => {
            const res = await api.post(`/organization/${id}/leave`);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['organization', id] });
        }
    });

    const requestMutation = useMutation({
        mutationFn: async (reason: string) => {
            const res = await api.post(`/organization/${id}/request`, { reason });
            return res.data;
        },
        onSuccess: () => {
            setShowRequestForm(false);
            setReason('');
            queryClient.invalidateQueries({ queryKey: ['organization', id] });
        },
        onError: (error) => {
            const message = isAxiosError(error) ? error.response?.data?.error : null;
            toast.error(message || t('requestFailed'));
        }
    });

    // Unlisted members are absent from `members`, so use the viewer flags.
    const isMember = org?.viewer_is_member ?? false;
    const isOrgAdmin = org?.viewer_is_admin ?? false;
    const requestPending = org?.viewer_request?.state === 'P';

    // Mirrors the API gate: create permission + org-admin membership (or the
    // edit-all bypass).
    const canCreateProblem = hasPerm(user, 'judge.create_organization_problem')
        && (isOrgAdmin || hasPerm(user, 'judge.edit_all_problem'));
    const canCreateContest = hasPerm(user, 'judge.create_private_contest')
        && (isOrgAdmin || hasPerm(user, 'judge.edit_all_contest'));
    const canCreateBlog = hasPerm(user, 'judge.edit_organization_post')
        && (isOrgAdmin || hasPerm(user, 'judge.edit_all_post'));

    const handleJoin = () => {
        if (!isAuthenticated) {
            router.push('/login');
            return;
        }
        joinMutation.mutate();
    };

    const handleLeave = () => {
        if (!isAuthenticated) {
            router.push('/login');
            return;
        }
        leaveMutation.mutate();
    };

    const handleRequest = (e: React.FormEvent) => {
        e.preventDefault();
        if (!reason.trim()) {
            return;
        }
        requestMutation.mutate(reason.trim());
    };

    if (orgLoading) {
        return (
            <div className="max-w-7xl mx-auto space-y-4 mt-4 pb-20">
                <Skeleton className="h-64 rounded-md" />
                <Skeleton className="h-96 rounded-md" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Building2 size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-semibold mb-2">{t('notFound')}</h2>
                <p className="text-muted-foreground mb-6">{t('notFoundDesc')}</p>
                <Link
                    href="/organizations"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    {t('backToOrganizations')}
                </Link>
            </div>
        );
    }

    const tabButton = (tab: OrgTab, icon: React.ReactNode, label: string) => (
        <button
            onClick={() => selectTab(tab)}
            className={cn(
                'flex items-center gap-2 px-4 py-3 rounded text-sm font-medium transition-all flex-1 justify-center whitespace-nowrap',
                activeTab === tab ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted'
            )}
        >
            {icon}
            {label}
        </button>
    );

    const currentQuery =
        activeTab === 'problems' ? problemsQuery : activeTab === 'contests' ? contestsQuery : activeTab === 'submissions' ? submissionsQuery : null;
    const needsTab = activeTab === 'problems' || activeTab === 'contests' || activeTab === 'submissions';
    const tabForbidden =
        needsTab && isAxiosError(currentQuery?.error) && currentQuery?.error.response?.status === 403;

    return (
        <div className="max-w-7xl mx-auto space-y-8 mt-4 pb-20">
            {/* Back Button */}
            <Link
                href="/organizations"
                className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                {t('backToOrganizations')}
            </Link>

            {/* Header Card */}
            <div className="bg-card border rounded-md p-4 md:p-6 shadow-sm overflow-hidden relative">
                <div className="absolute top-0 right-0 p-32 opacity-5 pointer-events-none">
                    <Building2 size={300} />
                </div>

                <div className="relative z-10">
                    <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6 mb-8">
                        <div className="flex items-center gap-6">
                            {org.logo_override_image ? (
                                <img src={org.logo_override_image} alt="" className="w-24 h-24 rounded-md object-cover bg-muted" />
                            ) : (
                                <div className="w-24 h-24 rounded-md bg-muted flex items-center justify-center text-primary font-semibold text-5xl">
                                    {org.name[0]?.toUpperCase()}
                                </div>
                            )}
                            <div>
                                <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter mb-2">{org.name}</h1>
                                {org.short_name && (
                                    <p className="text-xs font-mono text-muted-foreground ">
                                        {org.short_name}
                                    </p>
                                )}
                            </div>
                        </div>

                        <div className="flex flex-wrap gap-3">
                            {isOrgAdmin && (
                                <Link
                                    href={`/organization/${org.id}-${org.slug}/manage`}
                                    className="px-4 py-3 rounded-xl border font-medium hover:bg-muted transition-colors flex items-center gap-2"
                                >
                                    <Settings size={18} />
                                    {t('manage')}
                                </Link>
                            )}
                            {canCreateProblem && (
                                <Link
                                    href={`/organization/${org.id}-${org.slug}/problem/new`}
                                    className="px-4 py-3 rounded-xl border font-medium hover:bg-muted transition-colors flex items-center gap-2"
                                >
                                    <FilePlus size={18} />
                                    {t('newProblem')}
                                </Link>
                            )}
                            {canCreateContest && (
                                <Link
                                    href={`/organization/${org.id}-${org.slug}/contest/new`}
                                    className="px-4 py-3 rounded-xl border font-medium hover:bg-muted transition-colors flex items-center gap-2"
                                >
                                    <Trophy size={18} />
                                    {t('newContest')}
                                </Link>
                            )}
                            {canCreateBlog && (
                                <Link
                                    href={`/organization/${org.id}-${org.slug}/blog/new`}
                                    className="px-4 py-3 rounded-xl border font-medium hover:bg-muted transition-colors flex items-center gap-2"
                                >
                                    <PenLine size={18} />
                                    {t('newBlogPost')}
                                </Link>
                            )}
                            {isAuthenticated && !isMember && org.is_open && (
                                <button
                                    onClick={handleJoin}
                                    disabled={joinMutation.isPending}
                                    className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors flex items-center gap-2 disabled:opacity-50"
                                >
                                    <UserPlus size={18} />
                                    {t('joinOrganization')}
                                </button>
                            )}
                            {isMember && (
                                <button
                                    onClick={handleLeave}
                                    disabled={leaveMutation.isPending}
                                    className="px-6 py-3 rounded-xl bg-muted text-muted-foreground font-medium hover:bg-destructive/10 hover:text-destructive transition-colors flex items-center gap-2 disabled:opacity-50"
                                >
                                    <UserMinus size={18} />
                                    {t('leave')}
                                </button>
                            )}
                            {!org.is_open && !isMember && requestPending && (
                                <Badge className="px-4 py-2 text-xs font-medium bg-amber-500/10 text-amber-500 border border-amber-500/20 flex items-center gap-2">
                                    <UserCheck size={14} />
                                    {t('requestPending')}
                                </Badge>
                            )}
                            {!org.is_open && !isMember && !requestPending && isAuthenticated && (
                                <button
                                    onClick={() => setShowRequestForm(open => !open)}
                                    className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors flex items-center gap-2"
                                >
                                    <UserPlus size={18} />
                                    {t('requestJoin')}
                                </button>
                            )}
                            {!org.is_open && !isMember && !isAuthenticated && (
                                <Badge className="px-4 py-2 text-xs font-medium bg-muted text-muted-foreground border border-dashed">
                                    {t('inviteOnly')}
                                </Badge>
                            )}
                        </div>
                    </div>

                    {showRequestForm && !isMember && (
                        <form onSubmit={handleRequest} className="mb-6 rounded-md border border-dashed bg-muted/20 p-4 space-y-3">
                            <label htmlFor="join-reason" className="block text-sm font-medium">
                                {t('requestReason')}
                            </label>
                            <textarea
                                id="join-reason"
                                value={reason}
                                onChange={e => setReason(e.target.value)}
                                required
                                rows={4}
                                placeholder={t('requestReasonPlaceholder')}
                                className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            />
                            <div className="flex justify-end gap-3">
                                <button
                                    type="button"
                                    onClick={() => setShowRequestForm(false)}
                                    className="px-4 py-2 rounded-xl border font-medium hover:bg-muted transition-colors"
                                >
                                    {tCommon('cancel')}
                                </button>
                                <button
                                    type="submit"
                                    disabled={requestMutation.isPending || !reason.trim()}
                                    className="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                                >
                                    {t('sendRequest')}
                                </button>
                            </div>
                        </form>
                    )}

                    {org.about && (
                        <div className="prose prose-sm dark:prose-invert max-w-none bg-muted/30 rounded-md p-6 border border-dashed">
                            <p className="whitespace-pre-wrap text-muted-foreground">{org.about}</p>
                        </div>
                    )}

                    {/* Stats */}
                    <div className="grid grid-cols-2 md:grid-cols-6 gap-4 mt-8">
                        <Link
                            href={`/organization/${org.id}-${org.slug}/blog`}
                            className="bg-muted/30 rounded-md p-4 border border-dashed hover:border-primary/30 hover:bg-muted/50 transition-all group"
                        >
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <FileText size={16} />
                                <span className="text-xs font-medium ">{t('blog')}</span>
                            </div>
                            <div className="text-3xl font-semibold group-hover:text-primary transition-colors">{tCommon('view')}</div>
                        </Link>
                        <div className="bg-muted/30 rounded-md p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Users size={16} />
                                <span className="text-xs font-medium ">{t('members')}</span>
                            </div>
                            <div className="text-3xl font-semibold">{org.member_count}</div>
                        </div>
                        <div className="bg-muted/30 rounded-md p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Crown size={16} />
                                <span className="text-xs font-medium ">{t('admins')}</span>
                            </div>
                            <div className="text-3xl font-semibold">{org.admins?.length || 0}</div>
                        </div>
                        <div className="bg-muted/30 rounded-md p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <TrendingUp size={16} />
                                <span className="text-xs font-medium ">{t('pp')}</span>
                            </div>
                            <div className="text-lg font-semibold text-primary">{formatPoints(org.performance_points)}</div>
                        </div>
                        <div className="bg-muted/30 rounded-md p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Shield size={16} />
                                <span className="text-xs font-medium ">{tCommon('status')}</span>
                            </div>
                            <div className="text-lg font-semibold text-emerald-500">{org.is_open ? 'Open' : 'Private'}</div>
                        </div>
                        <div className="bg-muted/30 rounded-md p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Hash size={16} />
                                <span className="text-xs font-medium ">{t('idLabel')}</span>
                            </div>
                            <div className="text-lg font-semibold font-mono">{org.id}</div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex bg-card border p-1 rounded-md shadow-sm sticky top-4 z-10 overflow-x-auto">
                {tabButton('members', <Users size={16} />, t('members'))}
                {tabButton('admins', <Crown size={16} />, t('adminsCount', { count: org.admins?.length || 0 }))}
                {tabButton('problems', <Puzzle size={16} />, t('problems'))}
                {tabButton('contests', <Trophy size={16} />, t('contests'))}
                {tabButton('submissions', <List size={16} />, t('submissions'))}
            </div>

            {/* Tab Content */}
            <motion.div
                key={activeTab}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                className="space-y-4"
            >
                {activeTab === 'members' && (
                    org.members && org.members.length > 0
                        ? (
                            <MemberTable
                                members={org.members}
                                sort={memberSort}
                                onSort={column => setMemberSort(toggleSort(memberSort, column, 'desc'))}
                            />
                        )
                        : <EmptyState icon={<Users size={48} />} text={t('noMembers')} />
                )}
                {activeTab === 'admins' && (
                    org.admins && org.admins.length > 0
                        ? <MemberTable members={org.admins} />
                        : <EmptyState icon={<Crown size={48} />} text={t('noAdmins')} />
                )}

                {tabForbidden && <MembersOnly />}

                {activeTab === 'problems' && !tabForbidden && (
                    <>
                        {problemsQuery.isLoading ? (
                            <Skeleton className="h-72 rounded-md" />
                        ) : problemsQuery.data?.data.length ? (
                            <div className="bg-card border rounded-md overflow-hidden">
                                <div className="overflow-x-auto">
                                    <table className="w-full text-left border-collapse">
                                        <thead>
                                            <tr className="bg-muted/30 border-b">
                                                <SortableHeader
                                                    label={t('problem')}
                                                    className="px-6 py-4"
                                                    active={problemSort.sort === 'name'}
                                                    order={problemSort.order}
                                                    onClick={() => {
                                                        setProblemSort(toggleSort(problemSort, 'name', 'asc'));
                                                        setTabPage(1);
                                                    }}
                                                />
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('group')}</th>
                                                <SortableHeader
                                                    label={t('points')}
                                                    className="px-6 py-4 text-center w-28"
                                                    active={problemSort.sort === 'points'}
                                                    order={problemSort.order}
                                                    onClick={() => {
                                                        setProblemSort(toggleSort(problemSort, 'points', 'desc'));
                                                        setTabPage(1);
                                                    }}
                                                />
                                                <SortableHeader
                                                    label={t('solvedBy')}
                                                    className="px-6 py-4 text-center w-28"
                                                    active={problemSort.sort === 'user_count'}
                                                    order={problemSort.order}
                                                    onClick={() => {
                                                        setProblemSort(toggleSort(problemSort, 'user_count', 'desc'));
                                                        setTabPage(1);
                                                    }}
                                                />
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-28">{t('acRate')}</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-muted/50">
                                            {problemsQuery.data.data.map(p => (
                                                <tr key={p.code} className="hover:bg-muted/10 transition-colors">
                                                    <td className="px-6 py-4">
                                                        <Link href={`/problem/${p.code}`} className="inline-flex flex-col gap-0.5 hover:text-primary transition-colors">
                                                            <span className="font-medium text-sm">{p.name}</span>
                                                            <span className="text-xs uppercase font-mono text-muted-foreground opacity-40">{p.code}</span>
                                                        </Link>
                                                    </td>
                                                    <td className="px-6 py-4 text-sm text-muted-foreground">{p.group}</td>
                                                    <td className="px-6 py-4 text-center text-sm font-medium">{p.points}</td>
                                                    <td className="px-6 py-4 text-center text-sm">{p.user_count}</td>
                                                    <td className="px-6 py-4 text-center">
                                                        <span className={cn('text-sm font-medium', p.is_solved && 'text-emerald-500')}>
                                                            {formatPoints(p.ac_rate)}%
                                                        </span>
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        ) : (
                            <EmptyState icon={<Puzzle size={48} />} text={t('noProblems')} />
                        )}
                    </>
                )}

                {activeTab === 'contests' && !tabForbidden && (
                    <>
                        {contestsQuery.isLoading ? (
                            <Skeleton className="h-72 rounded-md" />
                        ) : contestsQuery.data?.data.length ? (
                            <div className="bg-card border rounded-md overflow-hidden">
                                <div className="overflow-x-auto">
                                    <table className="w-full text-left border-collapse">
                                        <thead>
                                            <tr className="bg-muted/30 border-b">
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('contest')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-48">{t('startTime')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('duration')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-28">{t('participants')}</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-muted/50">
                                            {contestsQuery.data.data.map(c => (
                                                <tr key={c.key} className="hover:bg-muted/10 transition-colors">
                                                    <td className="px-6 py-4">
                                                        <Link href={`/contest/${c.key}`} className="font-medium text-sm hover:text-primary transition-colors">
                                                            {c.name}
                                                        </Link>
                                                    </td>
                                                    <td className="px-6 py-4 text-center text-sm text-muted-foreground">
                                                        {dayjs(c.start_time).format('L HH:mm')}
                                                    </td>
                                                    <td className="px-6 py-4 text-center text-sm text-muted-foreground">
                                                        {dayjs(c.end_time).diff(dayjs(c.start_time), 'minute')}m
                                                    </td>
                                                    <td className="px-6 py-4 text-center text-sm">{c.user_count}</td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        ) : (
                            <EmptyState icon={<Trophy size={48} />} text={t('noContests')} />
                        )}
                    </>
                )}

                {activeTab === 'submissions' && !tabForbidden && (
                    <>
                        {submissionsQuery.isLoading ? (
                            <Skeleton className="h-72 rounded-md" />
                        ) : submissionsQuery.data?.data.length ? (
                            <div className="bg-card border rounded-md overflow-hidden">
                                <div className="overflow-x-auto">
                                    <table className="w-full text-left border-collapse">
                                        <thead>
                                            <tr className="bg-muted/30 border-b">
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground w-28">{tCommon('status')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('problem')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground w-40">{t('member')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-28">{t('language')}</th>
                                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-right w-36">{t('date')}</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-muted/50">
                                            {submissionsQuery.data.data.map(s => (
                                                <tr key={s.id} className="hover:bg-muted/10 transition-colors">
                                                    <td className="px-6 py-4">
                                                        <Link href={`/submission/${s.id}`} className="outline-none">
                                                            <Badge
                                                                variant={getStatusVariant(s.result || s.status)}
                                                                className="px-3 py-1 rounded text-xs font-medium uppercase"
                                                            >
                                                                {s.result || s.status}
                                                            </Badge>
                                                        </Link>
                                                    </td>
                                                    <td className="px-6 py-4">
                                                        <Link href={`/problem/${s.problem}`} className="inline-flex flex-col gap-0.5 hover:text-primary transition-colors">
                                                            <span className="font-medium text-sm">{s.problem_name || s.problem}</span>
                                                            <span className="text-xs uppercase font-mono text-muted-foreground opacity-40">{s.problem}</span>
                                                        </Link>
                                                    </td>
                                                    <td className="px-6 py-4">
                                                        <Link href={`/user/${s.user}`} className="text-sm font-medium hover:text-primary transition-colors">
                                                            {s.user}
                                                        </Link>
                                                    </td>
                                                    <td className="px-6 py-4 text-center text-xs font-medium text-muted-foreground">{s.language}</td>
                                                    <td className="px-6 py-4 text-right text-xs font-medium text-muted-foreground">
                                                        {dayjs(s.date).fromNow()}
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        ) : (
                            <EmptyState icon={<List size={48} />} text={t('noSubmissions')} />
                        )}
                    </>
                )}

                {needsTab && !tabForbidden && currentQuery && (
                    <PaginationBar
                        page={tabPage}
                        onPageChange={setTabPage}
                        total={currentQuery.data?.total}
                        pageSize={tabPageSize}
                        onPageSizeChange={size => { setTabPageSize(size); setTabPage(1); }}
                    />
                )}
            </motion.div>
        </div>
    );
}
