---
name: executing-tasks
description: Use when you have a written implementation task to execute in a separate session with review checkpoints
---

# Executing Plans

## Overview

Load task, review critically, execute all tasks, report when complete.

**Announce at start:** "I'm using the skillgrid:executing-tasks skill to implement this task."

**Note:** Tell your human partner that Superpowers works much better with access to subagents (Claude Code, Codex CLI, Codex App, Copilot CLI, and Gemini CLI all qualify; see the per-platform tool refs in `../using-skillgrid/references/`). If subagents are available, use skillgrid:subagent-driven-development instead of this skill.

## The Process

### Step 1: Load and Review Plan
1. Ensure an isolated workspace: use skillgrid:using-git-worktrees to create one or verify the existing one
2. Read task file
3. Review critically - identify any questions or concerns about the task
4. If concerns: Raise them with your human partner before starting
5. If no concerns: Create todos for the task items and proceed

### Step 2: Execute Tasks

For each task:
1. Mark as in_progress
2. Follow each step exactly (task has bite-sized steps)
3. Run verifications as specified
4. Mark as completed

### Step 3: Complete Development

After all tasks complete and verified:
- Announce: "I'm using the skillgrid:finishing-a-development-branch skill to complete this work."
- **REQUIRED SUB-SKILL:** Use skillgrid:finishing-a-development-branch
- Follow that skill to verify tests, present options, execute choice

## When to Stop and Ask for Help

**STOP executing immediately when:**
- Hit a blocker (missing dependency, test fails, instruction unclear)
- Plan has critical gaps preventing starting
- You don't understand an instruction
- Verification fails repeatedly

**Ask for clarification rather than guessing.**

## When to Revisit Earlier Steps

**Return to Review (Step 1) when:**
- Partner updates the task based on your feedback
- Fundamental approach needs rethinking

**Don't force through blockers** - stop and ask.

## Remember
- Review task critically first
- Follow task steps exactly
- Don't skip verifications
- Reference skills when task says to
- Stop when blocked, don't guess
- Never start implementation on main/master branch without explicit user consent
