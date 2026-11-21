# Common Rationalizations (skill authoring)

Load this when you (or the user) push back on extracting a script,
keeping a skill small, or following the deterministic boundary.

| Rationalization | Reality |
|---|---|
| "The script is overkill for a 5-line check" | Five lines of prose × every invocation = 5 lines of tokens × every invocation. A 15-line script is written once. The crossover is at ~3 invocations. |
| "The agent can figure out what to do from the description" | "Figure out" means re-derive the logic every time, with a risk of misreading. A script is a fixed, tested derivation. The agent's job is to *call* it, not to *be* it. |
| "I'll write the skill first and extract scripts later" | "Later" is when the skill is already bloated and the extraction is a refactor instead of a design decision. Script-first makes the boundary explicit at authoring time. |
| "This step is 'almost deterministic' — just one judgment call at the end" | Split it. The deterministic 90% goes to a script (JSON output); the 10% judgment stays in prose. A mixed step is two steps. |
| "The skill is only 350 lines, it's within budget" | Budget measures size, not correctness. A 300-line skill that re-interprets deterministic logic on every run is more expensive than a 150-line skill that calls three scripts. The budget is a floor, not a goal. |
| "The agent is smart enough to parse the JSON itself" | "Smart enough" is "usually, but not always." A script that parses the JSON and prints a verdict is deterministic. The agent parsing it is a coin flip with a 95% hit rate — and that 5% is the bug you chase at 2am. |
