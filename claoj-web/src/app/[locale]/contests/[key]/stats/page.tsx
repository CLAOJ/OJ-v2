'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { useParams } from 'next/navigation';

interface ProblemStats {
  problem_code: string;
  problem_name: string;
  problem_label: string;
  total: number;
  accepted: number;
  ac_rate: string;
}

interface LanguageStats {
  language: string;
  count: number;
}

interface ScoreDistribution {
  score_range: string;
  count: number;
}

interface ContestPublicStats {
  contest_key: string;
  contest_name: string;
  total_participants: number;
  total_submissions: number;
  problems: ProblemStats[];
  languages: LanguageStats[];
  score_distribution: ScoreDistribution[];
}

export default function ContestStatsPage() {
  const t = useTranslations();
  const params = useParams();
  const key = params.key as string;

  const [stats, setStats] = useState<ContestPublicStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchStats() {
      try {
        const response = await api.get(`/contest/${key}/stats/public`);
        setStats(response.data);
      } catch (error) {
        // Failed to fetch contest stats - will show error state
      } finally {
        setLoading(false);
      }
    }

    fetchStats();
  }, [key]);

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center text-red-500">
          <h1 className="text-2xl font-bold mb-2">Contest not found</h1>
          <p>The contest you're looking for doesn't exist or isn't visible.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">{stats.contest_name}</h1>
        <p className="text-gray-600 dark:text-gray-400">
          {t('stats.title')}
        </p>
      </div>

      {/* Overview Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <StatCard
          label={t('stats.total_participants')}
          value={stats.total_participants.toLocaleString()}
        />
        <StatCard
          label={t('stats.total_submissions')}
          value={stats.total_submissions.toLocaleString()}
        />
        <StatCard
          label={t('stats.total_problems')}
          value={stats.problems.length.toString()}
        />
        <StatCard
          label={t('stats.submissions_per_participant')}
          value={(stats.total_submissions / Math.max(stats.total_participants, 1)).toFixed(1)}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Problem Statistics */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.problem_statistics')}
          </h2>
          <div className="space-y-3">
            {stats.problems.map((problem) => (
              <div
                key={problem.problem_code}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded"
              >
                <div className="flex items-center gap-3">
                  <span className="font-bold text-blue-600 w-8">{problem.problem_label}</span>
                  <div>
                    <div className="font-medium">{problem.problem_name}</div>
                    <div className="text-sm text-gray-500">{problem.problem_code}</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-semibold">
                    {problem.accepted} / {problem.total}
                  </div>
                  <div className="text-sm text-gray-500">{problem.ac_rate}% AC</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Language Distribution */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.language_distribution')}
          </h2>
          <div className="space-y-3">
            {stats.languages.map((lang) => (
              <div
                key={lang.language}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded"
              >
                <span className="font-medium">{lang.language}</span>
                <div className="flex items-center gap-4">
                  <div className="w-32 bg-gray-200 dark:bg-gray-600 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full"
                      style={{
                        width: `${(lang.count / Math.max(stats.total_submissions, 1)) * 100}%`,
                      }}
                    />
                  </div>
                  <span className="font-semibold w-16 text-right">
                    {lang.count.toLocaleString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Score Distribution */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.score_distribution')}
          </h2>
          <div className="space-y-3">
            {stats.score_distribution.map((dist) => (
              <div
                key={dist.score_range}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded"
              >
                <span className="font-medium">{dist.score_range}</span>
                <div className="flex items-center gap-4">
                  <div className="w-32 bg-gray-200 dark:bg-gray-600 rounded-full h-2">
                    <div
                      className="bg-green-600 h-2 rounded-full"
                      style={{
                        width: `${(dist.count / Math.max(stats.total_participants, 1)) * 100}%`,
                      }}
                    />
                  </div>
                  <span className="font-semibold w-16 text-right">
                    {dist.count.toLocaleString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Top Performers Placeholder */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">
            {t('stats.top_performers')}
          </h2>
          <p className="text-gray-500 text-center py-8">
            {t('stats.view_ranking')}
          </p>
          <a
            href={`/contests/${key}`}
            className="block text-center mt-4 text-blue-600 hover:underline"
          >
            {t('contest.scoreboard')} →
          </a>
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
