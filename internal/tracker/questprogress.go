package tracker

// QuestStepStatus is a step plus its resolved completion state.
type QuestStepStatus struct {
	QuestStep
	Done    bool // complete (own flag set, or rolled up from a later step's flag)
	Known   bool // backed by a resolvable flag (its own or a later step's)
	Current bool // first incomplete required step — the "what to do next"
}

// QuestStatus is a quest plus its resolved per-step state.
type QuestStatus struct {
	Quest
	Phase     string // game phase of the starting region (for grouping in the UI)
	Steps     []QuestStepStatus
	Trackable bool // at least one step has a resolvable anchor (flag or item) in this save
}

// Done returns the number of completed required (non-optional) steps.
func (q QuestStatus) Done() int {
	n := 0
	for _, s := range q.Steps {
		if !s.Optional && s.Done {
			n++
		}
	}
	return n
}

// Total returns the number of required (non-optional) steps.
func (q QuestStatus) Total() int {
	n := 0
	for _, s := range q.Steps {
		if !s.Optional {
			n++
		}
	}
	return n
}

// Started reports whether any step is complete.
func (q QuestStatus) Started() bool {
	for _, s := range q.Steps {
		if s.Done {
			return true
		}
	}
	return false
}

// Complete reports whether every required step is complete.
func (q QuestStatus) Complete() bool { return q.Total() > 0 && q.Done() == q.Total() }

// CurrentStep returns the first incomplete required step (the next objective), if any.
func (q QuestStatus) CurrentStep() (QuestStepStatus, bool) {
	for _, s := range q.Steps {
		if s.Current {
			return s, true
		}
	}
	return QuestStepStatus{}, false
}

// QuestProgress is the full computed quest state, quests in progression order.
type QuestProgress struct {
	Quests []QuestStatus
}

// Totals returns overall completed and total required step counts.
func (p QuestProgress) Totals() (done, total int) {
	for _, q := range p.Quests {
		done += q.Done()
		total += q.Total()
	}
	return
}

// CategoryTotals splits step totals into base-game and DLC.
func (p QuestProgress) CategoryTotals() (baseDone, baseTotal, dlcDone, dlcTotal int) {
	for _, q := range p.Quests {
		if q.DLC {
			dlcDone += q.Done()
			dlcTotal += q.Total()
		} else {
			baseDone += q.Done()
			baseTotal += q.Total()
		}
	}
	return
}

// QuestsComplete returns the number of fully-complete quests and the total.
func (p QuestProgress) QuestsComplete() (complete, total int) {
	for _, q := range p.Quests {
		total++
		if q.Complete() {
			complete++
		}
	}
	return
}

// ComputeQuests builds quest progress for every quest from flagReader, which reports
// whether an NPC quest-progression event flag is set (the same reader the boss view
// uses for defeat flags).
//
// Completion rolls up monotonically: a step counts as done if its own flag is set OR
// any later step's flag is — so the first incomplete required step is always an
// accurate "what to do next", even where an intermediate beat has no dedicated flag.
// Because the flags are validated to be durable and monotonic, the roll-up never
// over-claims: the deepest flag sits on the final step, so a quest only reads as
// complete when it truly is.
func ComputeQuests(flagReader Defeater) QuestProgress {
	var out []QuestStatus
	for _, m := range orderQuests(allQuests) {
		qs := computeQuest(m.Quest, flagReader)
		qs.Phase = m.Phase
		out = append(out, qs)
	}
	return QuestProgress{Quests: out}
}

func computeQuest(q Quest, flagReader Defeater) QuestStatus {
	n := len(q.Steps)
	steps := make([]QuestStepStatus, n)

	// First pass: resolve each step's own flag.
	ownDone := make([]bool, n)
	ownKnown := make([]bool, n)
	for i, s := range q.Steps {
		steps[i] = QuestStepStatus{QuestStep: s}
		if s.Flag != 0 && flagReader != nil {
			if set, ok := flagReader(s.Flag); ok {
				ownKnown[i] = true
				ownDone[i] = set
			}
		}
	}

	// Backward pass: roll later completions/known-ness up to earlier steps.
	trackable := false
	laterDone, laterKnown := false, false
	for i := n - 1; i >= 0; i-- {
		steps[i].Done = ownDone[i] || laterDone
		steps[i].Known = ownKnown[i] || laterKnown
		if ownKnown[i] {
			trackable = true
		}
		laterDone = laterDone || ownDone[i]
		laterKnown = laterKnown || ownKnown[i]
	}

	// Forward pass: mark the first incomplete required step as the current objective.
	for i := range steps {
		if !steps[i].Optional && !steps[i].Done {
			steps[i].Current = true
			break
		}
	}

	return QuestStatus{Quest: q, Steps: steps, Trackable: trackable}
}
