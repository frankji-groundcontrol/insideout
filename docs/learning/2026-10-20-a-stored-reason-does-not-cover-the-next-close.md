# A stored completion reason does not cover the next close

## Lesson

A reason column left on a row is history, not a fresh decision. When a
roadmap node moves `in_progress` to `done` again after a reopen, the
sentence from the previous completion does not satisfy the rule. The
update that performs the transition has to carry a new one-sentence
reason, unless the node has linked evidence.

## Evidence

`TestRoadmapCompletion_RefusesDoneWithoutEvidenceOrReason` closes a node
with "The pilot shipped.", reopens it, and refuses a second close that
sends no reason. The listed row still holds the old sentence.

## When to apply

Any later "override with a reason" rule (overdue continue / abandon, a
second validation) should ask for the reason on the write that changes
state, not treat a leftover column as current consent.
