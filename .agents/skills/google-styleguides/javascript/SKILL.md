---
name: javascript
description: Google's official JavaScript style guide for ES6+. Covers const/let, arrow functions, template literals, destructuring, modules, naming conventions, JSDoc, and formatting. Enforces modern JavaScript patterns and best practices.
---

# Google JavaScript Style Guide

> Official Google JavaScript coding standards for ES6+ code.

## Golden Rules

1. **Use `const` by default** — `let` only when reassignment needed, never `var`
2. **Arrow functions for callbacks** — traditional functions for methods
3. **Template literals** for string interpolation
4. **Destructuring** where it improves readability
5. **Named exports** over default exports
6. **JSDoc for public APIs** — document parameters and return types
7. **2-space indentation** — consistent formatting

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Classes | UpperCamelCase | `UserService` |
| Functions/Methods | lowerCamelCase | `getUserById` |
| Variables | lowerCamelCase | `userCount` |
| Constants | UPPER_SNAKE_CASE | `MAX_SIZE` |
| Private fields | #privateField | `#userId` |
| Files | lower-kebab-case | `user-service.js` |

### Variables

```javascript
// ✓ CORRECT
const greeting = 'Hello';
let count = 0;

// ✗ INCORRECT
var greeting = 'Hello';  // never use var
```

### Functions

```javascript
// ✓ CORRECT - arrow functions for callbacks
const double = (x) => x * 2;
array.map(item => item.id);

// ✓ CORRECT - traditional functions for methods
class User {
  getName() {
    return this.name;
  }
}

// ✓ CORRECT - default parameters
function greet(name, greeting = 'Hello') {
  return `${greeting}, ${name}!`;
}
```

### Strings

```javascript
// ✓ CORRECT - template literals
const name = 'Alice';
const greeting = `Hello, ${name}!`;

// ✗ INCORRECT
const greeting = 'Hello, ' + name + '!';
```

### Destructuring

```javascript
// ✓ CORRECT - object destructuring
const {id, name} = user;
const {x, y, ...rest} = coordinates;

// ✓ CORRECT - array destructuring
const [first, second] = items;
const [head, ...tail] = list;

// ✓ CORRECT - function parameters
function processUser({id, name, email}) {
  // ...
}
```

### Modules

```javascript
// ✓ CORRECT - named exports
export function myFunction() { /* ... */ }
export class MyClass { /* ... */ }

// ✓ CORRECT - named imports
import {myFunction, MyClass} from './my-module.js';

// ✗ INCORRECT - default exports (avoid)
export default function() { /* ... */ }
```

### Classes

```javascript
// ✓ CORRECT
class Animal {
  #name;  // private field

  constructor(name) {
    this.#name = name;
  }

  speak() {
    return `${this.#name} makes a sound.`;
  }
}
```

### JSDoc

```javascript
/**
 * Fetches user data from the API.
 * @param {number} userId - The user ID to fetch.
 * @param {Object} options - Optional configuration.
 * @param {number} options.timeout - Request timeout in ms.
 * @return {Promise<Object>} The user data.
 */
function fetchUser(userId, {timeout = 5000} = {}) {
  // ...
}
```

### Promises and Async/Await

```javascript
// ✓ CORRECT - async/await
async function fetchUser(id) {
  const response = await fetch(`/api/users/${id}`);
  if (!response.ok) {
    throw new Error(`HTTP error: ${response.status}`);
  }
  return response.json();
}

// ✓ CORRECT - error handling
try {
  const user = await fetchUser(1);
} catch (error) {
  console.error('Failed to fetch user:', error);
}
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Using `var` | Use `const` / `let` |
| String concatenation | Use template literals |
| Default exports | Use named exports |
| Missing JSDoc | Document public APIs |
| Traditional functions for callbacks | Use arrow functions |
| Ignoring destructuring | Use where it helps readability |

## When to Use This Guide

- Writing new JavaScript code
- Refactoring existing JavaScript
- Code reviews
- Setting up ESLint rules
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/javascript
```

## Full Guide

See [javascript.md](javascript.md) for complete details, examples, and edge cases.
