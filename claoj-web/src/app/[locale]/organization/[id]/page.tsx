'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { OrganizationDetail, OrganizationMember } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useState, use } from 'react';
import {
    Building2,
    Users,
    UserPlus,
    UserMinus,
    Shield,
    Crown,
    TrendingUp,
    Hash,
    Trophy,
    MapPin,
    Calendar,
    ArrowLeft
} from 'lucide-react';
import { cn, getRankColor } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { motion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';

dayjs.extend(relativeTime);

export default function OrganizationDetailPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const t = useTranslations('Organization');
    const router = useRouter();
    const { user, loading } = useAuth();
    const [activeTab, setActiveTab] = useState<'members' | 'admins'>('members');
    const queryClient = useQueryClient();

    const isAuthenticated = !!user;

    const { data: org, isLoading: orgLoading } = useQuery({
        queryKey: ['organization', id],
        queryFn: async () => {
            const res = await api.get<OrganizationDetail>(`/organization/${id}`);
            return res.data;
        }
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

    const isMember = org?.members.some(m => m.username === user?.username);

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

    if (orgLoading) {
        return (
            <div className="max-w-7xl mx-auto space-y-8 mt-4 pb-20">
                <Skeleton className="h-64 rounded-[3rem]" />
                <Skeleton className="h-96 rounded-[3rem]" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Building2 size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-black mb-2">Organization Not Found</h2>
                <p className="text-muted-foreground mb-6">The organization you&apos;re looking for doesn&apos;t exist or has been removed.</p>
                <Link
                    href="/organizations"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    Back to Organizations
                </Link>
            </div>
        );
    }

    const MemberCard = ({ member, isAdmin }: { member: OrganizationMember; isAdmin: boolean }) => (
        <Link
            href={`/user/${member.username}`}
            className="group block bg-card border rounded-[2rem] p-6 hover:border-primary/30 hover:shadow-lg transition-all"
        >
            <div className="flex items-center gap-4">
                <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center text-primary font-black text-xl group-hover:scale-110 transition-transform">
                    {member.display_name?.[0]?.toUpperCase() || member.username[0]?.toUpperCase()}
                </div>
                <div className="flex-1">
                    <h4 className="font-black text-lg group-hover:text-primary transition-colors">
                        {member.display_name || member.username}
                    </h4>
                    <p className="text-[10px] font-mono text-muted-foreground">@{member.username}</p>
                    <div className="flex items-center gap-3 mt-2">
                        {isAdmin && (
                            <Badge className="text-[10px] font-black uppercase tracking-widest bg-amber-500/10 text-amber-500 border-amber-500/20">
                                <Crown size={12} className="inline mr-1" /> Admin
                            </Badge>
                        )}
                        <span className="text-[10px] text-muted-foreground font-bold">
                            Joined {dayjs(member.joined_at).fromNow()}
                        </span>
                    </div>
                </div>
                <div className="text-right">
                    <div className="text-2xl font-black text-primary">{member.role === 'admin' ? 'ADM' : 'MEM'}</div>
                </div>
            </div>
        </Link>
    );

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            {/* Back Button */}
            <Link
                href="/organizations"
                className="inline-flex items-center gap-2 text-sm font-bold text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                Back to Organizations
            </Link>

            {/* Header Card */}
            <div className="bg-card border rounded-[3rem] p-8 md:p-12 shadow-sm overflow-hidden relative">
                <div className="absolute top-0 right-0 p-32 opacity-5 pointer-events-none">
                    <Building2 size={300} />
                </div>

                <div className="relative z-10">
                    <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6 mb-8">
                        <div className="flex items-center gap-6">
                            <div className="w-24 h-24 rounded-[2rem] bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center text-primary font-black text-5xl shadow-lg shadow-primary/10">
                                {org.name[0]?.toUpperCase()}
                            </div>
                            <div>
                                <h1 className="text-4xl md:text-5xl font-black tracking-tighter mb-2">{org.name}</h1>
                                {org.short_name && (
                                    <p className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">
                                        {org.short_name}
                                    </p>
                                )}
                            </div>
                        </div>

                        <div className="flex gap-3">
                            {isAuthenticated && user && !isMember && org.is_open && (
                                <button
                                    onClick={handleJoin}
                                    disabled={joinMutation.isPending}
                                    className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors flex items-center gap-2 disabled:opacity-50"
                                >
                                    <UserPlus size={18} />
                                    Join Organization
                                </button>
                            )}
                            {isMember && (
                                <button
                                    onClick={handleLeave}
                                    disabled={leaveMutation.isPending}
                                    className="px-6 py-3 rounded-xl bg-muted text-muted-foreground font-bold hover:bg-destructive/10 hover:text-destructive transition-colors flex items-center gap-2 disabled:opacity-50"
                                >
                                    <UserMinus size={18} />
                                    Leave
                                </button>
                            )}
                            {!org.is_open && !isMember && (
                                <Badge className="px-4 py-2 text-[10px] font-black uppercase tracking-widest bg-muted text-muted-foreground border border-dashed">
                                    Invite Only
                                </Badge>
                            )}
                        </div>
                    </div>

                    {org.about && (
                        <div className="prose prose-sm dark:prose-invert max-w-none bg-muted/30 rounded-[2rem] p-6 border border-dashed">
                            <p className="whitespace-pre-wrap text-muted-foreground">{org.about}</p>
                        </div>
                    )}

                    {/* Stats */}
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-8">
                        <div className="bg-muted/30 rounded-[1.5rem] p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Users size={16} />
                                <span className="text-[10px] font-black uppercase tracking-widest">Members</span>
                            </div>
                            <div className="text-3xl font-black">{org.member_count}</div>
                        </div>
                        <div className="bg-muted/30 rounded-[1.5rem] p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Crown size={16} />
                                <span className="text-[10px] font-black uppercase tracking-widest">Admins</span>
                            </div>
                            <div className="text-3xl font-black">{org.admins?.length || 0}</div>
                        </div>
                        <div className="bg-muted/30 rounded-[1.5rem] p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Shield size={16} />
                                <span className="text-[10px] font-black uppercase tracking-widest">Status</span>
                            </div>
                            <div className="text-lg font-black text-emerald-500">{org.is_open ? 'Open' : 'Private'}</div>
                        </div>
                        <div className="bg-muted/30 rounded-[1.5rem] p-4 border border-dashed">
                            <div className="flex items-center gap-2 text-muted-foreground mb-2">
                                <Hash size={16} />
                                <span className="text-[10px] font-black uppercase tracking-widest">ID</span>
                            </div>
                            <div className="text-lg font-black font-mono">{org.id}</div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex bg-card border p-1 rounded-2xl shadow-sm sticky top-4 z-10 backdrop-blur-xl bg-card/80 overflow-x-auto">
                <button
                    onClick={() => setActiveTab('members')}
                    className={cn(
                        "flex items-center gap-2 px-6 py-3 rounded-xl text-sm font-bold transition-all flex-1 justify-center",
                        activeTab === 'members'
                            ? "bg-primary text-primary-foreground shadow-lg shadow-primary/20"
                            : "text-muted-foreground hover:bg-muted"
                    )}
                >
                    <Users size={16} />
                    Members ({org.members?.length || 0})
                </button>
                <button
                    onClick={() => setActiveTab('admins')}
                    className={cn(
                        "flex items-center gap-2 px-6 py-3 rounded-xl text-sm font-bold transition-all flex-1 justify-center",
                        activeTab === 'admins'
                            ? "bg-primary text-primary-foreground shadow-lg shadow-primary/20"
                            : "text-muted-foreground hover:bg-muted"
                    )}
                >
                    <Crown size={16} />
                    Admins ({org.admins?.length || 0})
                </button>
            </div>

            {/* Tab Content */}
            <motion.div
                key={activeTab}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                className="grid grid-cols-1 md:grid-cols-2 gap-4"
            >
                {activeTab === 'members' && (
                    <>
                        {org.members && org.members.length > 0 ? (
                            org.members.map((member) => (
                                <MemberCard key={member.id} member={member} isAdmin={false} />
                            ))
                        ) : (
                            <div className="col-span-full text-center py-20 text-muted-foreground border-2 border-dashed rounded-[3rem]">
                                <Users size={48} className="mx-auto mb-4 opacity-10" />
                                <p className="font-bold">No members yet</p>
                            </div>
                        )}
                    </>
                )}
                {activeTab === 'admins' && (
                    <>
                        {org.admins && org.admins.length > 0 ? (
                            org.admins.map((member) => (
                                <MemberCard key={member.id} member={member} isAdmin />
                            ))
                        ) : (
                            <div className="col-span-full text-center py-20 text-muted-foreground border-2 border-dashed rounded-[3rem]">
                                <Crown size={48} className="mx-auto mb-4 opacity-10" />
                                <p className="font-bold">No admins</p>
                            </div>
                        )}
                    </>
                )}
            </motion.div>
        </div>
    );
}
