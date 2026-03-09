'use client';

interface UserProfile {
    id: number;
    username: string;
}

interface Language {
    id: number;
    name: string;
    key: string;
}

interface AuthorsSectionProps {
    users?: { data: UserProfile[] };
    languages?: { data: Language[] };
    selectedAuthors: number[];
    selectedLangs: number[];
    onAuthorToggle: (userId: number, checked: boolean) => void;
    onLangToggle: (langId: number, checked: boolean) => void;
}

export function AuthorsSection({
    users,
    languages,
    selectedAuthors,
    selectedLangs,
    onAuthorToggle,
    onLangToggle
}: AuthorsSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">Authors & Languages</h3>

            <div>
                <label htmlFor="problem-authors" className="text-sm font-medium text-muted-foreground block mb-2">
                    Authors
                </label>
                <div id="problem-authors" className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-48 overflow-y-auto" role="group" aria-labelledby="problem-authors">
                    {users?.data.map(u => (
                        <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedAuthors.includes(u.id)}
                                onChange={(e) => onAuthorToggle(u.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm truncate">{u.username}</span>
                        </label>
                    ))}
                </div>
            </div>

            <div>
                <label htmlFor="allowed-languages" className="text-sm font-medium text-muted-foreground block mb-2">
                    Allowed Languages
                </label>
                <div id="allowed-languages" className="grid grid-cols-2 md:grid-cols-3 gap-2" role="group" aria-labelledby="allowed-languages">
                    {languages?.data.map(lang => (
                        <label key={lang.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedLangs.includes(lang.id)}
                                onChange={(e) => onLangToggle(lang.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm">{lang.name}</span>
                        </label>
                    ))}
                </div>
            </div>
        </div>
    );
}
