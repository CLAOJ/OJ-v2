// Value lists for judge_profile fields, matching v1 (judge/models/choices.py)
// so both sites accept what the other saves.

export const ACE_THEMES: [string, string][] = [
    ['auto', 'Follow site theme'],
    ['ambiance', 'Ambiance'], ['chaos', 'Chaos'], ['chrome', 'Chrome'], ['clouds', 'Clouds'],
    ['clouds_midnight', 'Clouds Midnight'], ['cobalt', 'Cobalt'], ['crimson_editor', 'Crimson Editor'],
    ['dawn', 'Dawn'], ['dreamweaver', 'Dreamweaver'], ['eclipse', 'Eclipse'], ['github', 'Github'],
    ['idle_fingers', 'Idle Fingers'], ['katzenmilch', 'Katzenmilch'], ['kr_theme', 'KR Theme'],
    ['kuroir', 'Kuroir'], ['merbivore', 'Merbivore'], ['merbivore_soft', 'Merbivore Soft'],
    ['mono_industrial', 'Mono Industrial'], ['monokai', 'Monokai'], ['pastel_on_dark', 'Pastel on Dark'],
    ['solarized_dark', 'Solarized Dark'], ['solarized_light', 'Solarized Light'], ['terminal', 'Terminal'],
    ['textmate', 'Textmate'], ['tomorrow', 'Tomorrow'], ['tomorrow_night', 'Tomorrow Night'],
    ['tomorrow_night_blue', 'Tomorrow Night Blue'], ['tomorrow_night_bright', 'Tomorrow Night Bright'],
    ['tomorrow_night_eighties', 'Tomorrow Night Eighties'], ['twilight', 'Twilight'],
    ['vibrant_ink', 'Vibrant Ink'], ['xcode', 'XCode'],
];

export const SITE_THEMES = ['auto', 'light', 'dark'] as const;

// IANA zones from the browser (v1 uses pytz's list; same names). Falls back to
// a minimal list on browsers without Intl.supportedValuesOf.
export function timezones(): string[] {
    try {
        return Intl.supportedValuesOf('timeZone');
    } catch {
        return ['UTC', 'Asia/Ho_Chi_Minh'];
    }
}

/** Keeps a stored value selectable even when it is not in the list. */
export function withCurrent(values: string[], current: string): string[] {
    return current && !values.includes(current) ? [current, ...values] : values;
}
