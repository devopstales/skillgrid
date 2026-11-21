# Google Style Guides Skill for AI Coding Agents

<div align="center">
  <p>Official Google style guides packaged as AI coding agent skills.</p>
</div>

<div align="center">
  <a href="https://github.com/testdino-hq/google-styleguides/stargazers">
    <img src="https://img.shields.io/github/stars/testdino-hq/google-styleguides?style=social" alt="GitHub Stars">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-CC--By%203.0-blue" alt="License">
  </a>
  <img src="https://img.shields.io/badge/languages-17-green" alt="17 Languages">
</div>

---

Give your AI coding agent — Cursor, Claude, GitHub Copilot, Windsurf, or any AI tool — instant access to Google's battle-tested style guides. These aren't just docs; they're production-ready coding standards used across Google's engineering organization.

> **17 language style guides** covering TypeScript, Python, JavaScript, Java, C++, Go, Swift, and more — formatted for AI agent consumption.

---

## Table of Contents

- [Who Is This For?](#who-is-this-for)
- [Why These Style Guides?](#why-these-style-guides)
- [Quick Start](#quick-start)
- [Available Style Guides](#available-style-guides)
- [How It Works](#how-it-works)
- [Contributing](#contributing)
- [License](#license)

---

## Who Is This For?

- **Developers using AI coding agents** who want consistent, Google-standard code generation
- **Teams adopting Google's style guides** across their codebase
- **Anyone tired of inconsistent AI-generated code** that doesn't follow best practices
- **Engineering managers** enforcing coding standards

---

## Why These Style Guides?

AI agents generate code that works but often violates style conventions. Wrong naming, inconsistent formatting, anti-patterns — the same issues, over and over.

These guides package Google's official style guides in a format AI agents can reference during code generation. The result: code that follows industry-standard conventions from the first keystroke.

Source: [https://google.github.io/styleguide/](https://google.github.io/styleguide/)

---

## Quick Start

The `skills` CLI copies style guide content into your project so AI coding agents can reference these standards when generating code.

### Install All Style Guides

```bash
npx skills add testdino-hq/google-styleguides-skills
```

### Install Individual Style Guides

Pick only the languages you need:

```bash
# Frontend
npx skills add testdino-hq/google-styleguides-skills/typescript
npx skills add testdino-hq/google-styleguides-skills/javascript
npx skills add testdino-hq/google-styleguides-skills/html-css
npx skills add testdino-hq/google-styleguides-skills/angularjs

# Backend
npx skills add testdino-hq/google-styleguides-skills/python
npx skills add testdino-hq/google-styleguides-skills/java
npx skills add testdino-hq/google-styleguides-skills/go
npx skills add testdino-hq/google-styleguides-skills/cpp
npx skills add testdino-hq/google-styleguides-skills/csharp

# Mobile
npx skills add testdino-hq/google-styleguides-skills/swift
npx skills add testdino-hq/google-styleguides-skills/objective-c

# Other
npx skills add testdino-hq/google-styleguides-skills/shell
npx skills add testdino-hq/google-styleguides-skills/r
npx skills add testdino-hq/google-styleguides-skills/common-lisp
npx skills add testdino-hq/google-styleguides-skills/vim-script
npx skills add testdino-hq/google-styleguides-skills/json
npx skills add testdino-hq/google-styleguides-skills/markdown
```

---

## Available Style Guides

| Language | Command | What's Covered |
| --- | --- | --- |
| **TypeScript** | `npx skills add testdino-hq/google-styleguides-skills/typescript` | Types, interfaces, naming, null handling, enums, imports |
| **JavaScript** | `npx skills add testdino-hq/google-styleguides-skills/javascript` | ES6+, modules, naming, formatting, JSDoc |
| **Python** | `npx skills add testdino-hq/google-styleguides-skills/python` | PEP 8, type hints, docstrings, imports, comprehensions |
| **Java** | `npx skills add testdino-hq/google-styleguides-skills/java` | Naming, formatting, Javadoc, exceptions, best practices |
| **C++** | `npx skills add testdino-hq/google-styleguides-skills/cpp` | Headers, naming, formatting, classes, memory management |
| **Go** | `npx skills add testdino-hq/google-styleguides-skills/go` | Formatting, naming, comments, error handling, concurrency |
| **Swift** | `npx skills add testdino-hq/google-styleguides-skills/swift` | Naming, optionals, protocols, error handling, formatting |
| **Objective-C** | `npx skills add testdino-hq/google-styleguides-skills/objective-c` | Naming, formatting, memory management, protocols |
| **C#** | `npx skills add testdino-hq/google-styleguides-skills/csharp` | Naming, formatting, LINQ, async/await, XML docs |
| **HTML/CSS** | `npx skills add testdino-hq/google-styleguides-skills/html-css` | Formatting, naming, semantics, accessibility |
| **AngularJS** | `npx skills add testdino-hq/google-styleguides-skills/angularjs` | Controllers, services, directives, modules |
| **Shell** | `npx skills add testdino-hq/google-styleguides-skills/shell` | Bash scripting, naming, error handling, portability |
| **R** | `npx skills add testdino-hq/google-styleguides-skills/r` | Naming, formatting, functions, documentation |
| **Common Lisp** | `npx skills add testdino-hq/google-styleguides-skills/common-lisp` | Naming, formatting, macros, documentation |
| **Vim Script** | `npx skills add testdino-hq/google-styleguides-skills/vim-script` | Plugin structure, naming, portability |
| **JSON** | `npx skills add testdino-hq/google-styleguides-skills/json` | Formatting, naming, structure, comments |
| **Markdown** | `npx skills add testdino-hq/google-styleguides-skills/markdown` | Formatting, structure, links, lists |

---

## How It Works

1. **Install the skill** using `npx skills add`
2. **The style guide is copied** to your project's `.kiro/skills/` directory
3. **Your AI agent reads it** when generating code in that language
4. **Generated code follows** Google's style conventions automatically

The skills system integrates with:
- Cursor
- Claude
- GitHub Copilot
- Windsurf
- Any AI coding tool that supports context injection

---

## Example: TypeScript

Before installing the skill:

```typescript
// AI-generated code without style guide
function getUser(id) {
  const user = users.find(u => u.id === id)!;
  return user;
}
```

After installing `testdino-hq/google-styleguides-skills/typescript`:

```typescript
// AI-generated code following Google TypeScript style
function getUser(id: number): User | undefined {
  return users.find(u => u.id === id);
}
```

---

## Language-Specific Quick Reference

### TypeScript
- Use `strict` mode
- Prefer interfaces over type aliases for objects
- Never use `any` — use `unknown`
- Explicit return types on public functions
- No non-null assertions (`!`)

### Python
- Follow PEP 8
- Type annotations on all public functions
- Google-style docstrings
- f-strings for formatting
- Comprehensions over `map()`/`filter()`

### JavaScript
- Use `const` by default
- Arrow functions for callbacks
- Template literals for strings
- Destructuring where appropriate
- JSDoc for public APIs

### Java
- UpperCamelCase for classes
- lowerCamelCase for methods
- Javadoc for public APIs
- Avoid wildcard imports
- Use Optional for nullable returns

### Go
- Run `gofmt` before commit
- Short variable names in small scopes
- Error handling, not exceptions
- Interfaces for abstraction
- Defer for cleanup

---

## Contributing

These guides are derived from [Google's official style guides](https://google.github.io/styleguide/). 

**About External Contributions:**  
With few exceptions, these style guides are copies of Google's internal style guides to assist developers working on Google-owned and originated open-source projects. Changes to the style guides are made to the internal style guides first and eventually copied into the versions found here.

**How to Contribute:**
- **Issues**: You can file issues using the [GitHub tracker](https://github.com/testdino-hq/google-styleguides/issues). Issues that raise questions, justify changes on technical merits, or point out obvious mistakes may get engagement.
- **Content Updates**: If you notice the content is outdated compared to the [official Google style guides](https://google.github.io/styleguide/), please open an issue.
- **Formatting Improvements**: Suggestions for better AI agent consumption are welcome.

**Note:** We are primarily optimizing for accurate representation of Google's style guides and effective AI agent integration.

---

## License

This project is licensed under the [MIT License](LICENSE).

Copyright (c) 2026 TestDino

See [LICENSE](LICENSE) for full details.

---

<div align="center">
  Built by <a href="https://testdino.com">TestDino</a> — production-grade code generation
</div>
