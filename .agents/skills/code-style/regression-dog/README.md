# regression-dog

A Claude Code skill that reviews code changes for unintended behavioral regressions.

## Install

```bash
npx skills add https://github.com/imaman/skills --skill regression-dog
```

## Usage

Start a fresh Claude Code session and ask it to review your changes:

```
/regression-dog HEAD~3
```

Or, in plain English:

```
flag all regressions in this branch
```

The skill also triggers from natural-language prompts about reviewing changes or checking for breakage, so the slash command is not the only way in.

## What you get

The output is split into two sections. Regressions come first, numbered and tagged with severity. Each one describes a behavioral delta: "this used to do X, now it does Y." After that, a Cleared section lists everything that was reviewed and found safe, so you can see the skill actually looked at your changes and not only at the parts it flagged.

## Tips

- Run it in a fresh session. Prior conversation context can bias the review toward things you've already discussed, which defeats the point.
- It works well on feature branches, not only refactors. New code can regress existing behavior through subtle interactions with the code it touches.
- If regressions show up, fix them and re-run until the list is clean.

## Background

Blog post with the story behind this skill: [20 Lines of Markdown Replaced My Code Review Bot](https://dev.to/itay-maman/20-lines-of-markdown-replaced-my-code-review-bot-26h2)
