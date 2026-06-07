package tracker

// satisfied reports whether the signal holds for the save: its event flag is set, or
// its item is owned.
func (s *QuestSignal) satisfied(flagReader Defeater, owner Owner) bool {
	if s == nil {
		return false
	}
	if s.Flag != 0 && flagReader != nil {
		if set, ok := flagReader(s.Flag); ok && set {
			return true
		}
	}
	if s.Item != 0 && owner != nil {
		if owner(Item{ID: s.Item, Kind: s.ItemKind}) {
			return true
		}
	}
	return false
}

// QuestStatus is a quest plus its resolved coarse state.
type QuestStatus struct {
	Quest
	Phase     string
	Complete  bool // completion signal satisfied
	Trackable bool // has a completion signal at all (else it's a reference guide)
}

// QuestProgress is the full computed quest state, quests in progression order.
type QuestProgress struct {
	Quests []QuestStatus
}

// ComputeQuests resolves each quest's coarse completion from two save-derived
// readers: flagReader (is this event flag set?, as the boss view uses) and owner (is
// this item owned?, as the items view uses). A quest with no completion signal is a
// guide (Trackable == false) and never reported complete.
func ComputeQuests(flagReader Defeater, owner Owner) QuestProgress {
	var out []QuestStatus
	for _, m := range orderQuests(allQuests) {
		st := QuestStatus{Quest: m.Quest, Phase: m.Phase, Trackable: m.CompleteWhen != nil}
		if st.Trackable {
			st.Complete = m.CompleteWhen.satisfied(flagReader, owner)
		}
		out = append(out, st)
	}
	return QuestProgress{Quests: out}
}

// Totals returns completed and trackable quest counts (guides are excluded — their
// completion can't be read from the save).
func (p QuestProgress) Totals() (complete, trackable int) {
	for _, q := range p.Quests {
		if q.Trackable {
			trackable++
			if q.Complete {
				complete++
			}
		}
	}
	return
}

// Counts returns completed, trackable, and total (including guides) quest counts.
func (p QuestProgress) Counts() (complete, trackable, total int) {
	complete, trackable = p.Totals()
	return complete, trackable, len(p.Quests)
}
