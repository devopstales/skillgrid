# Google JavaScript Style Guide

> Source: https://google.github.io/styleguide/jsguide.html

## Golden Rules

1. **Use `const` and `let`** — never `var`
2. **Use ES6+ features** — arrow functions, destructuring, template literals
3. **Semicolons are required** at end of every statement
4. **2-space indentation** — no tabs
5. **Single quotes** for strings (except JSON)
6. **Always use `===`** — never `==`
7. **No unused variables**

---

## 1. Variables

```javascript
// CORRECT
const PI = 3.14159;
let count = 0;

// INCORRECT
var name = 'Alice'; // never use var
```

### Destructuring

```javascript
const [first, second] = array;
const { name, age } = user;
const { name: userName } = user;  // with renaming
const { timeout = 5000 } = options;  // with defaults
```

---

## 2. Strings

```javascript
// CORRECT - single quotes
const name = 'Alice';

// CORRECT - template literals for interpolation
const greeting = `Hello, ${name}!`;

// INCORRECT
const greeting = 'Hello, ' + name + '!'; // use template literals
```

---

## 3. Functions

```javascript
// CORRECT - named function declaration
function processData(data) {
  return data.filter(Boolean);
}

// CORRECT - arrow functions for callbacks
const doubled = numbers.map(n => n * 2);

// CORRECT - default parameters
function greet(name, greeting = 'Hello') {
  return `${greeting}, ${name}!`;
}

// CORRECT - rest parameters
function sum(...numbers) {
  return numbers.reduce((a, b) => a + b, 0);
}
```

---

## 4. Classes

```javascript
// CORRECT
class Animal {
  #name; // private field

  constructor(name) {
    this.#name = name;
  }

  speak() {
    return `${this.#name} makes a sound.`;
  }
}

// INCORRECT
function Animal(name) { // use class syntax
  this.name = name;
}
```

---

## 5. Modules

```javascript
// CORRECT - named exports
export function processData(data) { return data; }
export const MAX_SIZE = 100;

// CORRECT - imports
import { processData } from './data-processor.js';
```

---

## 6. Arrays

```javascript
// CORRECT - array methods over loops
const evens = numbers.filter(n => n % 2 === 0);
const doubled = numbers.map(n => n * 2);
const total = numbers.reduce((acc, n) => acc + n, 0);

// CORRECT - spread
const combined = [...arr1, ...arr2];
const copy = [...original];
```

---

## 7. Async/Await

```javascript
// CORRECT
async function fetchUser(id) {
  const response = await fetch(`/api/users/${id}`);
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  return response.json();
}

// CORRECT - parallel operations
const [users, posts] = await Promise.all([fetchUsers(), fetchPosts()]);
```

---

## 8. Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Variables/Functions | lowerCamelCase | getUserById |
| Classes | UpperCamelCase | UserService |
| Constants | UPPER_SNAKE_CASE | MAX_RETRIES |
| Private fields | #name (ES2022) | #privateField |
| Files | lower-kebab-case | user-service.js |

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| `var` | Use `const`/`let` |
| `==` equality | Use `===` strict equality |
| String concatenation | Use template literals |
| `.bind(this)` | Use arrow functions |
| `arguments` object | Use rest parameters |
| `for` loops | Use `for...of` or array methods |
| Missing semicolons | Add semicolons |
