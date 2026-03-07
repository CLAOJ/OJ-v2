'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';

interface OverallStats {
  total_users: number;
  total_problems: number;
  total_submissions: number;
  total_contests: number;
  total_organizations: number;
  active_judges: number;
}

interface SubmissionStats {
  by_result: { result: string; count: number }[];
  by_language: { language: string; count: number; ac_count: number; ac_rate: number }[];
}

interface JudgeStats {
  judges: { name: string; online: boolean; last_ping: string; load: number }[];
  queue_size: number;
}

interface UserStats {
  total: number;
  active: number;
  banned: number;
  unverified: number;
}

export default function StatsPage() {
  const t = useTranslations();

  const [overallStats, setOverallStats] = useState<OverallStats | null>(null);
  const [submissionStats, setSubmissionStats] = useState<SubmissionStats | null>(null);
  const [judgeStats, setJudgeStats] = useState<JudgeStats | null>(null);
  const [userStats, setUserStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchStats() {
      try {
        const [overall, submissions, judges, users] = await Promise.all([
          api.get('/stats/overall'),
          api.get('/stats/submissions'),
          api.get('/stats/judges'),
          api.get('/stats/users'),
        ]);

        setOverallStats(overall.data);
        setSubmissionStats(submissions.data);
        setJudgeStats(judges.data);
        setUserStats(users.data);
      } catch (error) {
        // Failed to fetch stats - will show error state
      } finally {
        setLoading(false);
      }
    }

    fetchStats();
  }, []);

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">{t('stats.title')}</h1>

      {/* Overall Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
        <StatCard
          label={t('stats.total_users')}
          value={overallStats?.total_users.toLocaleString() ?? '-'}
        />
        <StatCard
          label={t('stats.total_problems')}
          value={overallStats?.total_problems.toLocaleString() ?? '-'}
        />
        <StatCard
          label={t('stats.total_submissions')}
          value={overallStats?.total_submissions.toLocaleString() ?? '-'}
        />
        <StatCard
          label={t('stats.total_contests')}
          value={overallStats?.total_contests.toLocaleString() ?? '-'}
        />
        <StatCard
          label={t('stats.total_organizations')}
          value={overallStats?.total_organizations.toLocaleString() ?? '-'}
        />
        <StatCard
          label={t('stats.active_judges')}
          value={overallStats?.active_judges.toLocaleString() ?? '-'}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Submission Stats by Result */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.submissions_by_result')}
          </h2>
          <div className="space-y-2">
            {submissionStats?.by_result.map((stat) => (
              <div key={stat.result} className="flex justify-between items-center">
                <span className="font-mono">{stat.result}</span>
                <span className="font-semibold">{stat.count.toLocaleString()}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Language Stats */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.submissions_by_language')}
          </h2>
          <div className="space-y-2">
            {submissionStats?.by_language.slice(0, 10).map((stat) => (
              <div key={stat.language} className="flex justify-between items-center">
                <span>{stat.language}</span>
                <div className="text-right">
                  <div className="font-semibold">{stat.count.toLocaleString()}</div>
                  <div className="text-sm text-gray-500">
                    {stat.ac_rate.toFixed(1)}% AC
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* User Stats */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.user_statistics')}
          </h2>
          <div className="space-y-4">
            <UserStatRow
              label={t('stats.total_users')}
              value={userStats?.total.toLocaleString() ?? '-'}
            />
            <UserStatRow
              label={t('stats.active_users')}
              value={userStats?.active.toLocaleString() ?? '-'}
            />
            <UserStatRow
              label={t('stats.banned_users')}
              value={userStats?.banned.toLocaleString() ?? '-'}
            />
            <UserStatRow
              label={t('stats.unverified_users')}
              value={userStats?.unverified.toLocaleString() ?? '-'}
            />
          </div>
        </div>

        {/* Judge Stats */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.judge_queue')}
          </h2>
          <div className="mb-4">
            <div className="flex justify-between items-center mb-2">
              <span>{t('stats.queue_size')}</span>
              <span className="font-semibold text-lg">
                {judgeStats?.queue_size.toLocaleString() ?? '-'}
              </span>
            </div>
          </div>
          <h3 className="font-semibold mb-2">{t('stats.judges')}</h3>
          <div className="space-y-2 max-h-48 overflow-y-auto">
            {judgeStats?.judges.map((judge) => (
              <div
                key={judge.name}
                className="flex justify-between items-center p-2 bg-gray-50 dark:bg-gray-700 rounded"
              >
                <div className="flex items-center gap-2">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      judge.online ? 'bg-green-500' : 'bg-gray-400'
                    }`}
                  />
                  <span className="font-mono">{judge.name}</span>
                </div>
                <div className="text-sm text-gray-500">
                  {judge.load}% load
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 text-center">
      <div className="text-2xl font-bold text-blue-600 mb-1">{value}</div>
      <div className="text-sm text-gray-600 dark:text-gray-400">{label}</div>
    </div>
  );
}

function UserStatRow({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex justify-between items-center">
      <span className="text-gray-600 dark:text-gray-400">{label}</span>
      <span className="font-semibold">{value}</span>
    </div>
  );
}
