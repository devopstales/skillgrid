---
version: alpha
name: JSM Engineering Workflow (reconstructed)
description: Reconstructed design draft for jsmastery.com — dark editorial terminal aesthetic for the nine-phase agent-skill workflow.
colors:
  primary: "#6EE7A0"
  background: "#020013"
  surface: "#050709"
  elevated: "#111214"
  panel: "#08090a"
  border: "#2E3238"
  button-border: "#282A2B"
  foreground: "#FFFFFF"
  foreground-muted: "#E2E8F4"
  foreground-body: "#D4DDF1"
  foreground-tertiary: "#8A8F98"
  foreground-faint: "#9EAABF"
typography:
  display:
    fontFamily: Geist Sans
    fontSize: 64px
    fontWeight: 700
    lineHeight: 65.28px
    letterSpacing: -2.6px
  heading-xl:
    fontFamily: Geist Sans
    fontSize: 48px
    fontWeight: 500
    lineHeight: 48px
    letterSpacing: -2px
  heading-lg:
    fontFamily: Geist Sans
    fontSize: 32px
    fontWeight: 500
    lineHeight: 38px
    letterSpacing: -1px
  heading-md:
    fontFamily: Geist Sans
    fontSize: 26px
    fontWeight: 500
    lineHeight: 32px
    letterSpacing: -0.5px
  lead:
    fontFamily: Geist Sans
    fontSize: 20px
    fontWeight: 400
    lineHeight: 28px
  body:
    fontFamily: Geist Sans
    fontSize: 18px
    fontWeight: 400
    lineHeight: 28px
  body-sm:
    fontFamily: Geist Sans
    fontSize: 14px
    fontWeight: 400
    lineHeight: 21px
  label:
    fontFamily: Geist Mono
    fontSize: 12px
    fontWeight: 400
    lineHeight: 16px
  caption:
    fontFamily: Geist Mono
    fontSize: 10px
    fontWeight: 400
    lineHeight: 13px
  command:
    fontFamily: Geist Mono
    fontSize: 14px
    fontWeight: 400
    lineHeight: 22px
  cta:
    fontFamily: Geist Mono
    fontSize: 18px
    fontWeight: 700
    lineHeight: 24px
    letterSpacing: 1px
rounded:
  chip: 99px
  card: 7px
  pill: 9999px
spacing:
  base: 16px
  sm: 18px
  gutter-mobile: 16px
  gutter-desktop: 80px
  content-top: 40px
  content-bottom: 120px
components:
  cta-primary:
    backgroundColor: "{colors.foreground}"
    textColor: "#000000"
    rounded: "{rounded.pill}"
    padding: 0px 30px
    height: 52px
  cta-secondary:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.foreground-muted}"
    rounded: "{rounded.pill}"
    padding: 0px 30px
    height: 52px
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.card}"
    padding: 24px
  doc-title:
    textColor: "{colors.foreground-tertiary}"
    typography: "{typography.heading-xl}"
  terminal-panel:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.foreground-faint}"
    rounded: "{rounded.card}"
  skill-section:
    backgroundColor: "rgba(110, 231, 160, 0.08)"
    textColor: "{colors.foreground-body}"
    rounded: "{rounded.chip}"
---

# Overview

Reconstructed design draft for jsmastery.com, the home of the nine-phase JSM Engineering Workflow. The presentation is a dark, editorial terminal aesthetic: an almost-black background with white and cool-grey type, a single mint-green accent, and monospace set in for every technical artifact (slash-commands, file paths, labels, install commands). Depth is conveyed by flat tonal layering, not shadow.

# Colors

A near-black cool base with a single mint-green driver. Green is reserved for command syntax and the skill/command system; it is not a general-purpose accent.

- **Primary (#6EE7A0):** The sole accent. Terminal command text, skill and owner tags, and a 0.08-opacity wash used to highlight relevant skill-reference sections.
- **Background (#020013):** A very dark indigo-black page base (body).
- **Surface (#050709):** Slightly warmer black used by the sticky nav, footer, and skill cards.
- **Elevated (#111214):** Dark pill for the secondary CTA and raised controls.
- **Panel (#08090A):** Mock-terminal window background.
- **Border (#2E3238):** 1px hairline for cards, the pipeline divider, and the mock-terminal frame.
- **Foreground (#FFFFFF):** Headlines and primary text.
- **Foreground muted (#E2E8F4):** Secondary CTA label and lighter body text.
- **Foreground body (#D4DDF1):** Cool grey for body copy and footer links.
- **Foreground tertiary (#8A8F98):** Weakened heading tone on doc-template titles and section kickers.
- **Foreground faint (#9EAABF):** Phase numbers, table headers, and metadata.

# Typography

Two typefaces: **Geist Sans** for narrative display and body, **Geist Mono** for all technical content. Display and headings are tight (negative letter-spacing); mono is the default for labels, commands, paths, and install strings.

- **Display (64px / 700 / -2.6px):** Homepage hero headline only.
- **Heading scale (48 / 32 / 26px / 500):** Section and page titles; the doc-template H1 is set in the tertiary foreground tone.
- **Lead & body (20px / 18px):** Narrative copy in Geist Sans.
- **Mono roles:** label (12px), caption (10px), command (14px) for terminal lines and install strings; CTA (18px / 700 / uppercase) is the mono label on the primary button.

# Layout

Content is centered in a max-width column with generous side gutters (16px mobile, 80px desktop). Doc templates use a 40px top / 120px bottom rhythm to leave room for the sticky nav. A 1px green-to-transparent top gradient marks active skill-reference sections.

# Elevation & Depth

Flat tonal layering: the background is a dark indigo-black; nav, footer, and cards step to a warmer near-black (#050709); raised controls step to #111214. No drop shadows are used. The mock terminal is a #08090A panel framed by a #2E3238 hairline.

# Shapes

Pills rule: the primary and secondary CTAs are fully rounded (9999px), as are skill tags and install toggles (99px). Content cards and mock panels are square (0px) or lightly rounded (6–7px) with a 1px hairline border — the sharp, terminal-window frame.

# Components

- **Primary CTA:** White pill, black mono label, fully rounded, 52px tall, 30px horizontal padding (e.g. "INSTALL THE SKILLS").
- **Secondary CTA:** Dark pill (#111214) with a 1px #282A2B inset border and a lighter sans label, fully rounded, 52px tall.
- **Skill reference tag:** Mono, green (#6EE7A0), on a 0.08-opacity green wash.

# Do's and Don'ts

- Do reserve the green (#6EE7A0) accent for command syntax, skill tags, and the highlight wash only.
- Do set every technical string — slash-commands, file paths, labels, install commands — in Geist Mono.
- Do keep depth to flat tonal layering; do not add drop shadows.
- Don't use more than one accent color; the palette is single-accent green on a near-black base.
