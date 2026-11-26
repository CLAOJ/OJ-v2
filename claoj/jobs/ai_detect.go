package jobs

// Thresholds for AI submission detection (plagiarism check, Tasks 4-6).
const (
	aiRefsPerLang        = 5   // generated reference solutions kept per problem x language
	aiStatementMinChars  = 300 // shorter statements are too thin to generate references from
	aiMatchPct           = 50  // percent of a submission's lines matched to a reference to flag it
	aiTrivialShare       = 0.5 // share of trivial lines; above this a match does not count
	rewriteMaxSimilarity = 0.3 // max similarity to the original for a rewrite to count
	rewriteMinLines      = 5   // min lines a rewritten solution needs to be judged
	rewritesToFlag       = 2   // rewrites needed before a submission is flagged
	historyMargin        = 1.5 // margin over the contestant's own history
)
