/** Empty number input → undefined, so the mappers can tell "cleared" from zero. */
export const optionalInt = {
    setValueAs: (v: string) => (v === '' || v == null ? undefined : Number(v)),
};