'use client';

interface Problem {
    id: number;
    code: string;
    name: string;
    points: number;
}

interface ProblemsSectionProps {
    problems?: { data: Problem[] };
    selectedProblems: number[];
    onProblemToggle: (problemId: number, checked: boolean) => void;
}

export function ProblemsSection({ problems, selectedProblems, onProblemToggle }: ProblemsSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">Problems</h3>
            <p className="text-sm text-muted-foreground">
                Select problems to include in the contest. You can reorder them after creation.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2 max-h-64 overflow-y-auto">
                {problems?.data.map(p => (
                    <label key={p.id} className="flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30">
                        <input
                            type="checkbox"
                            checked={selectedProblems.includes(p.id)}
                            onChange={(e) => onProblemToggle(p.id, e.target.checked)}
                            className="rounded w-5 h-5"
                        />
                        <div className="flex-1 min-w-0">
                            <div className="font-medium text-sm truncate">{p.name}</div>
                            <div className="text-xs text-muted-foreground">
                                {p.code} • {p.points} pts
                            </div>
                        </div>
                    </label>
                ))}
            </div>
            {selectedProblems.length > 0 && (
                <div className="text-sm text-muted-foreground">
                    Selected: {selectedProblems.length} problem(s)
                </div>
            )}
        </div>
    );
}
