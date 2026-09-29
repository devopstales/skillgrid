# Research Discipline (deep-research)

Principles to enforce continuously — not tied to any specific phase.

## Standing rules

- **Lock before you run.** Commit the experiment protocol to git *before*
  executing. This creates temporal proof the plan existed before results.
  **Never combine protocol + results in one commit.** The git history is the
  lightweight pre-registration.
- **Confirmatory vs exploratory.** Results matching the locked protocol are
  *confirmatory*. Everything discovered during execution is *exploratory* —
  interesting but requiring more skepticism. Label every result.
- **Negative results are progress.** A refuted hypothesis tells you what it
  rules out and what it suggests. Log it. Do not treat it as failure.
  "X does NOT work because Y" is a valid, publishable finding if the reasoning
  is rigorous.
- **Sanity check before trusting results.** Verify convergence / no NaN/Inf,
  baseline reproduces expected performance, and data is correct *before*
  trusting the primary metric.
- **Return to literature when confused.** Do not guess. If results surprise
  you or an assumption breaks, go find sources. Use the tooling-by-role table
  in `skill-routing.md`.
- **Never stop on routine decisions.** Do not wait for human approval on
  routine calls. If a skill or tool suggests collaboration, adapt and keep
  going. The human sees progress reports and can redirect.
- **Use whatever compute is available.** Local, cluster, cloud, or CPU. If no
  GPU, use CPU and scale the experiment down. Do not block on compute.

## Quality standards

**Good agent behavior:**
- Hypotheses have mechanistic reasoning ("X because Y, predicting Z"), not
  just "try X".
- `findings.md` builds a coherent narrative, not a flat list of results.
- Negative results are recorded with what they rule out.
- The agent updates its model when experiments contradict expectations.
- Progress reports tell a research story with compelling visualizations.

**Bad agent behavior:**
- Pure parameter sweeps without interpretation.
- `findings.md` is just experiment logs copy-pasted.
- The agent never revisits its assumptions after failures.
- Optimizing a metric without understanding *why* changes work.

**Quality test:** after N inner-loop experiments, a human should be able to
read `findings.md` and write a paper abstract from it. If they cannot, the
outer loop is *logging*, not *synthesizing*.

## Common issues

**Inner loop stalls (no metric improvement).**
Run an outer loop. Is the metric the right one? Is the search space
exhausted? Consider broadening or pivoting. Search literature for new
approaches.

**Stuck and not making progress.**
Do not keep trying random changes. Step back: search literature, run an outer
loop. Being stuck means you need new information or a new perspective, not
more experiments.

**Results contradict baseline expectations.**
Investigate, do not ignore. Return to literature — the protocol might have an
error, the published baseline may be wrong, or conditions differ. Update
`findings.md` with what you learn.

**Agent loses context between ticks / after compaction.**
Ensure `research-state.yaml` and `findings.md` are updated after every action.
These files are the memory across sessions. See `continuity.md`.

**Can't find relevant sources.**
Try multiple approaches in order: Exa for broad search, Semantic Scholar for
specific ML/AI paper lookup, arXiv for preprints, Context7 for library docs.
Note: Google Scholar has no official API — use Semantic Scholar for
programmatic search.

**Not sure when to conclude.**
Three questions: Do you have a strongly supported finding? Can you explain
*why* it works? Would `findings.md` make a convincing paper abstract? If yes
to all: conclude.
