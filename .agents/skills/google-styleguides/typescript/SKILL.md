---
name: typescript
description: Google's official TypeScript style guide. Covers strict mode, type annotations, interfaces vs types, null handling, naming conventions, imports, and common mistakes. Enforces explicit return types, readonly properties, and avoidance of any/non-null assertions.
---

# Google TypeScript Style Guide

> Official Google TypeScript coding standards for consistent, maintainable code.

## Golden Rules

1. **Use TypeScript strictly** — enable `strict` mode in `tsconfig.json`
2. **Prefer interfaces over type aliases** for object shapes
3. **Never use `any`** — use `unknown` when type is truly unknown
4. **Use `const` by default**, `let` when reassignment needed, never `var`
5. **Avoid non-null assertions (`!`)** — handle nullability explicitly
6. **Use explicit return types** on public functions and methods
7. **Prefer `readonly`** for properties that should not change

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Classes/Interfaces | UpperCamelCase | `UserService`, `User` |
| Functions/Methods | lowerCamelCase | `getUserById` |
| Variables | lowerCamelCase | `userCount` |
| Constants | UPPER_SNAKE_CASE | `MAX_SIZE` |
| Enums | UpperCamelCase | `Direction` |
| Enum members | UPPER_SNAKE_CASE or UpperCamelCase | `Direction.UP` or `Status.Active` |
| Files | lower-kebab-case | `user-service.ts` |
| Test files | lower-kebab-case_test | `user-service_test.ts` |

### Type System

```typescript
// ✓ CORRECT - interfaces for object shapes
interface User {
  id: number;
  name: string;
}

// ✓ CORRECT - type aliases for unions
type StringOrNumber = string | number;

// ✗ INCORRECT - avoid any
const data: any = fetchData();

// ✓ CORRECT - use unknown and narrow
const data: unknown = fetchData();
if (typeof data === 'string') {
  console.log(data.toUpperCase());
}
```

### Null Handling

```typescript
// ✓ CORRECT - explicit nullability
function getUser(id: number): User | undefined {
  return users.find(u => u.id === id);
}

const user = getUser(1);
if (user !== undefined) {
  console.log(user.name);
}

// ✗ INCORRECT - non-null assertion
const user = getUser(1)!;
```

### Functions

```typescript
// ✓ CORRECT - explicit return types for public API
function add(a: number, b: number): number {
  return a + b;
}

// ✓ CORRECT - arrow functions for callbacks
const double = (x: number): number => x * 2;

// ✓ CORRECT - optional parameters with defaults
function greet(name: string, greeting = 'Hello'): string {
  return `${greeting}, ${name}!`;
}
```

### Imports

```typescript
// ✓ CORRECT - named exports
export function myFunction() { /* ... */ }
export interface MyInterface { id: number; }

// ✓ CORRECT - type-only imports
import type { MyInterface } from './types';
import { myFunction } from './my-module';

// ✗ INCORRECT - wildcard imports
import * as everything from './module';
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Using `any` type | Use `unknown` + type narrowing |
| Non-null assertion `!` | Check for null explicitly |
| `var` declarations | Use `const` / `let` |
| Missing return types | Add explicit return types |
| Wildcard imports | Use named imports |
| Mutable class properties | Use `readonly` when appropriate |
| Type aliases for objects | Use interfaces instead |

## tsconfig.json Recommended Settings

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  }
}
```

## When to Use This Guide

- Writing new TypeScript code
- Refactoring existing TypeScript
- Code reviews
- Setting up linting rules
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/typescript
```

## Full Guide

See [typescript.md](typescript.md) for complete details, examples, and edge cases.
