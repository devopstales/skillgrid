# Google TypeScript Style Guide

> **Source:** https://google.github.io/styleguide/tsguide.html
> 
> This guide is based on Google's internal TypeScript style guide. It provides comprehensive rules for writing consistent, maintainable TypeScript code.

## Table of Contents

1. [Introduction](#introduction)
2. [Source File Basics](#source-file-basics)
3. [Source File Structure](#source-file-structure)
4. [Imports and Exports](#imports-and-exports)
5. [Language Features](#language-features)
6. [Type System](#type-system)
7. [Naming](#naming)
8. [Comments and Documentation](#comments-and-documentation)
9. [Policies](#policies)

---

## Introduction

### Terminology

This guide uses RFC 2119 terminology:
- **must** / **must not** - Required / Forbidden
- **should** / **should not** - Recommended / Discouraged (prefer/avoid)
- **may** - Optional

All examples are non-normative and illustrate the rules but are not the only valid way to write code.

---

## Source File Basics

### File Encoding

Source files must be encoded in **UTF-8**.

### Whitespace Characters

- Only ASCII horizontal space (0x20) is allowed
- All other whitespace in strings must be escaped
- Use special escape sequences (`\'`, `\"`, `\\`, `\b`, `\f`, `\n`, `\r`, `\t`, `\v`) instead of numeric escapes

```typescript
// ✓ GOOD
const units = 'μs';
const output = '\ufeff' + content;  // byte order mark with comment

// ✗ BAD
const units = '\u03bcs'; // Hard to read
const output = '\ufeff' + content; // No explanation
```

---

## Source File Structure

Files consist of the following, **in order**, separated by exactly one blank line:

1. Copyright information (if present)
2. `@fileoverview` JSDoc (if present)
3. Imports (if present)
4. The file's implementation

### @fileoverview JSDoc

```typescript
/**
 * @fileoverview Description of file. Lorem ipsum dolor sit amet, consectetur
 * adipiscing elit, sed do eiusmod tempor incididunt.
 */
```

---

## Imports and Exports

### Import Types

Four variants of imports:

```typescript
// Module import (namespace)
import * as foo from '...';

// Named import (destructuring)
import {SomeThing} from '...';

// Default import (only when required by external code)
import SomeThing from '...';

// Side-effect import (only for libraries with side effects)
import '...';
```

### Import Paths

- Use relative paths (`./foo`) for files within the same project
- Limit parent steps (`../../../`) as they make structure hard to understand

```typescript
import {Symbol1} from 'path/from/root';
import {Symbol2} from '../parent/file';
import {Symbol3} from './sibling';
```

### Namespace vs Named Imports

- **Prefer named imports** for frequently used symbols or symbols with clear names
- **Prefer namespace imports** when using many symbols from large APIs

```typescript
// ✗ BAD: overlong import
import {Item as TableviewItem, Header as TableviewHeader, 
  Row as TableviewRow} from './tableview';

// ✓ GOOD: use namespace
import * as tableview from './tableview';
let item: tableview.Item|undefined;

// ✓ GOOD: named imports for common functions
import {describe, it, expect} from './testing';
```

### Exports

**Always use named exports. Never use default exports.**

```typescript
// ✓ GOOD
export class Foo { ... }
export function bar() { ... }
export const BAZ = 1;

// ✗ BAD
export default class Foo { ... }
```

**Why no default exports?**
- No canonical name (can be imported as anything)
- Harder to maintain
- Don't error when importing non-existent members
- Encourage putting everything in one object

### Export Visibility

- Only export symbols used outside the module
- Minimize exported API surface

### Mutable Exports

**Do not use `export let`**. Use explicit getter functions instead.

```typescript
// ✗ BAD
export let foo = 3;
setTimeout(() => { foo = 4; }, 1000);

// ✓ GOOD
let foo = 3;
setTimeout(() => { foo = 4; }, 1000);
export function getFoo() { return foo; }
```

### Container Classes

**Do not create container classes** with static methods/properties for namespacing.

```typescript
// ✗ BAD
export class Container {
  static FOO = 1;
  static bar() { return 1; }
}

// ✓ GOOD
export const FOO = 1;
export function bar() { return 1; }
```

### Import and Export Type

Use `import type` when importing symbols only as types:

```typescript
import type {Foo} from './foo';
import {Bar} from './foo';

// Or inline
import {type Foo, Bar} from './foo';
```

Use `export type` when re-exporting types:

```typescript
export type {AnInterface} from './foo';
```

### Modules Not Namespaces

**Do not use `namespace`**. Use ES6 modules with `import`/`export`.

```typescript
// ✗ BAD
namespace Rocket {
  function launch() { ... }
}

// ✗ BAD
/// <reference path="..."/>

// ✗ BAD
import x = require('mydep');

// ✓ GOOD
import {launch} from './rocket';
```

---

## Language Features

### Local Variable Declarations

**Use `const` and `let`. Never use `var`.**

```typescript
// ✓ GOOD
const foo = otherValue;  // Use if never reassigned
let bar = someValue;     // Use if reassigned later

// ✗ BAD
var foo = someValue;     // var has confusing function scope
```

Variables must not be used before their declaration.

**One variable per declaration:**

```typescript
// ✓ GOOD
let a = 1;
let b = 2;

// ✗ BAD
let a = 1, b = 2;
```

### Array Literals

**Do not use the `Array()` constructor:**

```typescript
// ✗ BAD
const a = new Array(2);      // [undefined, undefined]
const b = new Array(2, 3);   // [2, 3] - confusing!

// ✓ GOOD
const a = [2];
const b = [2, 3];
const c = [];
c.length = 2;
Array.from<number>({length: 5}).fill(0);  // [0, 0, 0, 0, 0]
```

**Do not define properties on arrays.** Use `Map` or `Object` instead.

**Spread Syntax:**

```typescript
// ✓ GOOD
const foo = [1];
const foo2 = [...foo, 6, 7];
const foo3 = [5, ...foo];

// ✗ BAD
const bar = [5, ...(shouldUseFoo && foo)]; // might be undefined
```

**Array Destructuring:**

```typescript
// ✓ GOOD
const [a, b, c, ...rest] = generateResults();
let [, b,, d] = someArray;  // Skip unused elements

// ✓ GOOD - default values
function destructured([a = 4, b = 2] = []) { … }

// ✗ BAD
function badDestructuring([a, b] = [4, 2]) { … }
```

### Object Literals

**Do not use the `Object()` constructor.** Use object literals (`{}` or `{a: 0}`).

**Iterating Objects:**

Do not use unfiltered `for...in`:

```typescript
// ✗ BAD
for (const x in someObj) {
  // x could come from prototype chain!
}

// ✓ GOOD
for (const x in someObj) {
  if (!someObj.hasOwnProperty(x)) continue;
  // now x is definitely on someObj
}

// ✓ BETTER
for (const x of Object.keys(someObj)) { ... }
for (const [key, value] of Object.entries(someObj)) { ... }
```

**Spread Syntax:**

```typescript
// ✓ GOOD
const foo = {num: 1};
const foo2 = {...foo, num: 5};  // foo2.num === 5
const foo3 = {num: 5, ...foo};  // foo3.num === 1

// ✗ BAD
const bar = {num: 5, ...(shouldUseFoo && foo)}; // might be undefined
```

**Object Destructuring:**

```typescript
// ✓ GOOD
interface Options {
  num?: number;
  str?: string;
}

function destructured({num, str = 'default'}: Options = {}) {}

// ✗ BAD - nested too deeply
function nestedTooDeeply({x: {num, str}}: {x: Options}) {}

// ✗ BAD - non-trivial default
function nontrivialDefault({num, str}: Options = {num: 42, str: 'default'}) {}
```

### Classes

**Class Declarations:**

Do not terminate with semicolons:

```typescript
// ✓ GOOD
class Foo {
}

// ✗ BAD
class Foo {
};
```

But statements with class expressions need semicolons:

```typescript
// ✓ GOOD
export const Baz = class extends Bar {
  method(): number { return this.x; }
};
```

**Method Declarations:**

Do not use semicolons between methods:

```typescript
// ✓ GOOD
class Foo {
  doThing() {
    console.log("A");
  }

  getOtherThing(): number {
    return 4;
  }
}

// ✗ BAD
class Foo {
  doThing() {
    console.log("A");
  };  // unnecessary semicolon
}
```

**Static Methods:**

- Avoid private static methods (prefer module-local functions)
- Do not rely on dynamic dispatch of static methods
- Avoid `this` in static contexts

**Constructors:**

Use parentheses even with no arguments:

```typescript
// ✓ GOOD
const x = new Foo();

// ✗ BAD
const x = new Foo;
```

Omit empty constructors or those that only call `super()`:

```typescript
// ✗ BAD - unnecessary
class UnnecessaryConstructor {
  constructor() {}
}

// ✓ GOOD - has parameter properties
class ParameterProperties {
  constructor(private myService) {}
}
```

**Class Members:**

**Do not use `#private` fields.** Use TypeScript's `private` instead:

```typescript
// ✗ BAD
class Clazz {
  #ident = 1;
}

// ✓ GOOD
class Clazz {
  private ident = 1;
}
```

**Use `readonly`** for properties never reassigned:

```typescript
class Foo {
  private readonly barService: BarService;
  
  constructor(barService: BarService) {
    this.barService = barService;
  }
}

// ✓ BETTER - parameter properties
class Foo {
  constructor(private readonly barService: BarService) {}
}
```

**Field Initializers:**

Initialize where declared when possible:

```typescript
// ✗ BAD
class Foo {
  private readonly userList: string[];
  constructor() {
    this.userList = [];
  }
}

// ✓ GOOD
class Foo {
  private readonly userList: string[] = [];
}
```

**Getters and Setters:**

- Getters must be pure functions (no side effects)
- Use to restrict visibility of internal details
- At least one accessor must be non-trivial

```typescript
// ✓ GOOD
class Foo {
  private wrappedBar = '';
  
  get bar() {
    return this.wrappedBar || 'bar';
  }
  
  set bar(wrapped: string) {
    this.wrappedBar = wrapped.trim();
  }
}

// ✗ BAD - pass-through accessors
class Bar {
  private barInternal = '';
  
  get bar() { return this.barInternal; }
  set bar(value: string) { this.barInternal = value; }
}
```

**Visibility:**

- Limit visibility as much as possible
- Never use `public` modifier except for non-readonly parameter properties
- Consider converting private methods to non-exported functions

```typescript
// ✗ BAD
class Foo {
  public bar = new Bar();
  constructor(public readonly baz: Baz) {}
}

// ✓ GOOD
class Foo {
  bar = new Bar();
  constructor(public baz: Baz) {}
}
```

### Functions

**Prefer function declarations** for named functions:

```typescript
// ✓ GOOD
function foo() {
  return 42;
}

// ✗ BAD
const foo = () => 42;
```

**Do not use function expressions.** Use arrow functions:

```typescript
// ✓ GOOD
bar(() => { this.doSomething(); })

// ✗ BAD
bar(function() { ... })
```

**Arrow Function Bodies:**

Use concise bodies for expressions, block bodies otherwise:

```typescript
// ✓ GOOD - concise body when return value used
const longThings = myValues.filter(v => v.length > 1000);

// ✓ GOOD - block body when return value unused
myPromise.then(v => {
  console.log(v);
});

// ✗ BAD - concise body when return value unused
myPromise.then(v => console.log(v));
```

**Rebinding `this`:**

Avoid rebinding `this`. Use arrow functions or explicit parameters:

```typescript
// ✗ BAD
function clickHandler() {
  this.textContent = 'Hello';
}
document.body.onclick = clickHandler;

// ✓ GOOD
document.body.onclick = () => { 
  document.body.textContent = 'hello'; 
};
```

**Arrow Functions as Properties:**

Avoid arrow function properties in classes:

```typescript
// ✗ BAD
class DelayHandler {
  private patienceTracker = () => {
    this.waitedPatiently = true;
  }
}

// ✓ GOOD
class DelayHandler {
  constructor() {
    setTimeout(() => {
      this.patienceTracker();
    }, 5000);
  }
  
  private patienceTracker() {
    this.waitedPatiently = true;
  }
}
```

**Parameter Initializers:**

Keep initializers simple with no side effects:

```typescript
// ✓ GOOD
function process(name: string, extraContext: string[] = []) {}
function activate(index = 0) {}

// ✗ BAD - side effect
let globalCounter = 0;
function newId(index = globalCounter++) {}
```

**Rest and Spread:**

Use rest parameters instead of `arguments`:

```typescript
// ✓ GOOD
function variadic(array: string[], ...numbers: number[]) {}

// Use spread instead of Function.prototype.apply
myFunction(...array, ...iterable);
```

### Primitive Literals

**String Literals:**

Use single quotes (`'`) for ordinary strings:

```typescript
// ✓ GOOD
const greeting = 'Hello';

// ✗ BAD
const greeting = "Hello";
```

**No line continuations:**

```typescript
// ✗ BAD
const LONG_STRING = 'This is a very long string. \
    It has problems.';

// ✓ GOOD
const LONG_STRING = 'This is a very long string. ' +
    'It uses concatenation.';
```

**Template Literals:**

Use template literals for complex concatenation:

```typescript
// ✓ GOOD
function arithmetic(a: number, b: number) {
  return `Here is a table:
${a} + ${b} = ${a + b}
${a} - ${b} = ${a - b}`;
}
```

**Number Literals:**

Use `0x`, `0o`, `0b` prefixes (lowercase) for hex, octal, binary.

### Type Coercion

Use `String()`, `Boolean()`, `!!`, or template literals for coercion:

```typescript
// ✓ GOOD
const bool = Boolean(false);
const str = String(aNumber);
const bool2 = !!str;
const str2 = `result: ${bool2}`;

// ✗ BAD
const str = '' + aNumber;  // Don't use + for coercion
```

**Enum to Boolean:**

Do not convert enums to booleans. Compare explicitly:

```typescript
enum SupportLevel { NONE, BASIC, ADVANCED }

// ✗ BAD
const level: SupportLevel = ...;
let enabled = Boolean(level);

// ✓ GOOD
let enabled = level !== SupportLevel.NONE;
```

**Parsing Numbers:**

Use `Number()` and check for `NaN`:

```typescript
// ✓ GOOD
const aNumber = Number('123');
if (!isFinite(aNumber)) throw new Error(...);

// ✗ BAD
const x = +y;  // Unary plus is easy to miss
const n = parseInt(someString, 10);  // Ignores trailing chars
```

### Control Structures

**Always use braces:**

```typescript
// ✓ GOOD
for (let i = 0; i < x; i++) {
  doSomethingWith(i);
}

if (x) {
  doSomething();
}

// ✗ BAD
if (x) doSomething();
for (let i = 0; i < x; i++) doSomethingWith(i);

// Exception: one-line if
if (x) x.doFoo();
```

**Iterating Containers:**

Prefer `for...of` for arrays:

```typescript
// ✓ GOOD
for (const x of someArr) {
  // x is a value
}

for (const [i, x] of someArr.entries()) {
  // i is index, x is value
}

// Also OK
for (let i = 0; i < someArr.length; i++) {
  const x = someArr[i];
}
```

Use `for...in` only for dict-style objects with `hasOwnProperty` check:

```typescript
// ✓ GOOD
for (const key in obj) {
  if (!obj.hasOwnProperty(key)) continue;
  doWork(key, obj[key]);
}

// ✓ BETTER
for (const key of Object.keys(obj)) {
  doWork(key, obj[key]);
}
```

### Exception Handling

**Instantiate errors with `new`:**

```typescript
// ✓ GOOD
throw new Error('Foo is not valid');

// ✗ BAD
throw Error('Foo is not valid');
```

**Only throw errors:**

```typescript
// ✗ BAD
throw 'oh noes!';
Promise.reject('oh noes!');

// ✓ GOOD
throw new Error('oh noes!');
Promise.reject(new Error('oh noes!'));
```

**Empty catch blocks:**

Explain why in a comment:

```typescript
// ✓ GOOD
try {
  return handleNumericResponse(response);
} catch (e: unknown) {
  // Response is not numeric. Continue to handle as text.
}
return handleTextResponse(response);
```

### Switch Statements

Must contain a `default` (even if empty):

```typescript
// ✓ GOOD
switch (x) {
  case Y:
    doSomething();
    break;
  default:
    // nothing to do
}
```

No fall-through (except empty cases):

```typescript
// ✓ GOOD
switch (x) {
  case X:
  case Y:
    doSomething();
    break;
  default:
}

// ✗ BAD
switch (x) {
  case X:
    doSomething();
    // fall through - not allowed!
  case Y:
    doOther();
}
```

### Equality Checks

**Always use `===` and `!==`:**

```typescript
// ✓ GOOD
if (foo === 'bar' || baz !== bam) { }

// ✗ BAD
if (foo == 'bar' || baz != bam) { }

// Exception: comparing to null
if (foo == null) {
  // Matches both null and undefined
}
```

### Type Assertions

**Avoid type assertions.** They are unsafe and don't insert runtime checks.

```typescript
// ✗ BAD
(x as Foo).foo();
y!.bar();

// ✓ GOOD - use runtime checks
if (x instanceof Foo) {
  x.foo();
}

if (y) {
  y.bar();
}
```

If necessary, add a comment explaining why it's safe:

```typescript
// x is a Foo because [reason]
(x as Foo).foo();
```

**Type assertion syntax:**

Use `as`, not angle brackets:

```typescript
// ✗ BAD
const x = (<Foo>z).length;

// ✓ GOOD
const x = (z as Foo).length;
```

**Object literals:**

Use type annotations, not assertions:

```typescript
interface Foo {
  bar: number;
  baz?: string;
}

// ✗ BAD
const foo = {
  bar: 123,
  bam: 'abc',  // Typo not caught!
} as Foo;

// ✓ GOOD
const foo: Foo = {
  bar: 123,
  bam: 'abc',  // Error: bam not on Foo
};
```

---

## Type System

### Type Inference

Code may rely on type inference for all type expressions.

**Leave out trivially inferred types:**

```typescript
// ✗ BAD
const x: boolean = true;
const y: Set<string> = new Set();

// ✓ GOOD
const x = true;
const y = new Set<string>();  // Generic needs explicit type
```

**Use annotations for complex expressions:**

```typescript
// Hard to infer
const value = await rpc.getSomeValue().transform();

// ✓ BETTER
const value: string[] = await rpc.getSomeValue().transform();
```

### Return Types

Return type annotations are optional but recommended for:
- Public APIs
- Complex return types
- Preventing future type errors

```typescript
// Optional but helpful
function add(a: number, b: number): number {
  return a + b;
}
```

### Undefined and Null

- TypeScript supports both `undefined` and `null`
- No general preference for one over the other
- Use what the API expects (Map uses `undefined`, DOM uses `null`)

**Nullable type aliases:**

Do not include `|null` or `|undefined` in type aliases:

```typescript
// ✗ BAD
type CoffeeResponse = Latte|Americano|undefined;

class CoffeeService {
  getLatte(): CoffeeResponse { ... }
}

// ✓ GOOD
type CoffeeResponse = Latte|Americano;

class CoffeeService {
  getLatte(): CoffeeResponse|undefined { ... }
}
```

**Prefer optional over `|undefined`:**

```typescript
// ✓ GOOD
interface CoffeeOrder {
  sugarCubes: number;
  milk?: Whole|LowFat;  // Optional, not |undefined
}

function pourCoffee(volume?: Milliliter) { ... }
```

### Structural Types

TypeScript uses structural typing (not nominal).

**Use interfaces for structural types:**

```typescript
// ✓ GOOD
interface Foo {
  a: number;
  b: string;
}

const foo: Foo = {
  a: 123,
  b: 'abc',
};

// ✗ BAD - relies on inference
const badFoo = {
  a: 123,
  b: 'abc',
};
```

**Prefer interfaces over type literals:**

```typescript
// ✓ GOOD
interface User {
  firstName: string;
  lastName: string;
}

// ✗ BAD
type User = {
  firstName: string,
  lastName: string,
}
```

### Array<T> Type

**Use `T[]` for simple types:**

```typescript
// ✓ GOOD
let a: string[];
let b: readonly string[];
let c: ns.MyObj[];
let d: string[][];

// ✗ BAD
let a: Array<string>;
let b: ReadonlyArray<string>;
```

**Use `Array<T>` for complex types:**

```typescript
// ✓ GOOD
let e: Array<{n: number, s: string}>;
let f: Array<string|number>;
let g: ReadonlyArray<string|number>;
```

### Indexable Types

Use for associative arrays, but consider `Map` or `Set` instead:

```typescript
// OK but consider Map
const fileSizes: {[fileName: string]: number} = {};
fileSizes['readme.txt'] = 541;

// ✓ BETTER - use meaningful labels
const users: {[userName: string]: number} = {};

// ✓ BEST - use Map
const users = new Map<string, number>();
```

### Mapped and Conditional Types

Use sparingly. They can make code hard to understand:

```typescript
// Consider if this is clearer than explicit interfaces
type FoodPreferences = Pick<User, 'favoriteIcecream'|'favoriteChocolate'>;

// Often better to be explicit
interface FoodPreferences {
  favoriteIcecream: string;
  favoriteChocolate: string;
}
```

### `any` Type

**Avoid `any`.** Consider alternatives:

1. **Provide a more specific type:**

```typescript
// ✓ GOOD
interface MyUserJson {
  name: string;
  email: string;
}

type MyType = number|string;
```

2. **Use `unknown`:**

```typescript
// ✓ GOOD
const val: unknown = value;
if (typeof val === 'string') {
  // Now can use as string
}

// ✗ BAD
const danger: any = value;
danger.whoops();  // Unchecked!
```

3. **Suppress with comment:**

```typescript
// This test only needs partial BookService
// tslint:disable-next-line:no-any
const mockBookService = ({get() { return mockBook; }} as any) as BookService;
```

### `{}` Type

**Avoid `{}` type.** Use instead:
- `unknown` - for any value including null/undefined
- `Record<string, T>` - for dictionary-like objects
- `object` - excludes primitives

### Tuple Types

Use tuples instead of Pair interfaces:

```typescript
// ✗ BAD
interface Pair {
  first: string;
  second: string;
}

// ✓ GOOD
function splitInHalf(input: string): [string, string] {
  return [x, y];
}

const [leftHalf, rightHalf] = splitInHalf('my string');
```

For clarity, consider inline object types:

```typescript
function splitHostPort(address: string): {host: string, port: number} {
  ...
}

const {host, port} = splitHostPort(userAddress);
```

### Wrapper Types

**Never use wrapper types** `String`, `Boolean`, `Number`, `Object`.

```typescript
// ✗ BAD
const s = new String('hello');
const b = new Boolean(false);

// ✓ GOOD
const s: string = 'hello';
const b: boolean = false;
```

Always use lowercase `string`, `boolean`, `number`, `object`.

---

## Naming

### Identifiers

Use only ASCII letters, digits, underscores (for constants and test names), and rarely `$`.

### Naming Style

**Do not decorate names with type information:**

- No trailing/leading underscores for private (use `private` keyword)
- No `opt_` prefix for optional parameters
- No `I` prefix for interfaces (unless idiomatic)
- Observables may use `$` suffix (team decision)

### Descriptive Names

Names must be clear to new readers:

```typescript
// ✓ GOOD
errorCount
dnsConnectionIndex
referrerUrl
customerId

// ✗ BAD
n                   // Meaningless
nErr                // Ambiguous
wgcConnections      // Only your group knows
cstmrId             // Deletes letters
kSecondsPerDay      // Hungarian notation
customerID          // Wrong camelCase
```

### Camel Case

Treat abbreviations as whole words:

```typescript
// ✓ GOOD
loadHttpUrl
XMLHttpRequest  // Platform name exception

// ✗ BAD
loadHTTPURL
```

### Rules by Identifier Type

| Style | Category |
|-------|----------|
| UpperCamelCase | class, interface, type, enum, decorator, type parameters |
| lowerCamelCase | variable, parameter, function, method, property, module alias |
| CONSTANT_CASE | global constant values, enum values |
| #ident | Never use private identifiers |

### Constants

Use `CONSTANT_CASE` for:
- Module-level constants
- Static readonly class properties
- Enum values

```typescript
const UNIT_SUFFIXES = {
  'milliseconds': 'ms',
  'seconds': 's',
};

class Foo {
  private static readonly MY_SPECIAL_NUMBER = 5;
}
```

**Not for local variables:**

```typescript
// ✗ BAD - local variable
function foo() {
  const SOME_CONSTANT = 5;  // Use lowerCamelCase
}
```

### Aliases

Match the format of the source:

```typescript
const {BrewStateEnum} = SomeType;
const CAPACITY = 5;

class Teapot {
  readonly BrewStateEnum = BrewStateEnum;
  readonly CAPACITY = CAPACITY;
}
```

---

## Comments and Documentation

### Comment Types

- `/** JSDoc */` - For documentation (users of the code)
- `//` - For implementation (only for code maintainers)

### JSDoc General Form

```typescript
/**
 * Multiple lines of JSDoc text are written here,
 * wrapped normally.
 * @param arg A number to do something to.
 */
function doSomething(arg: number) { … }

/** This short jsdoc describes the function. */
function doSomething(arg: number) { … }
```

### Markdown in JSDoc

JSDoc is written in Markdown:

```typescript
/**
 * Computes weight based on three factors:
 *
 * - items sent
 * - items received
 * - last timestamp
 */
```

### JSDoc Tags

Most tags must occupy their own line:

```typescript
// ✓ GOOD
/**
 * @param left A description of the left param.
 * @param right A description of the right param.
 */
function add(left: number, right: number) { ... }

// ✗ BAD
/**
 * @param left @param right
 */
function add(left: number, right: number) { ... }
```

### Document Top-Level Exports

Use JSDoc for all exported symbols:

```typescript
/** Component that prints "bar". */
@Component({
  selector: 'foo',
  template: 'bar',
})
export class FooComponent {}
```

### Method and Function Comments

- Omit if obvious from name and signature
- Start with verb phrase in third person
- Document parameter properties with `@param`

```typescript
/**
 * POSTs the request to start coffee brewing.
 * @param amountLitres The amount to brew. Must fit the pot size!
 */
brew(amountLitres: number, logger: Logger) { ... }
```

### Parameter Properties

```typescript
/** This class demonstrates parameter properties. */
class ParamProps {
  /**
   * @param percolator The percolator used for brewing.
   * @param beans The beans to brew.
   */
  constructor(
    private readonly percolator: Percolator,
    private readonly beans: CoffeeBean[]) {}
}
```

### JSDoc Type Annotations

**Do not use JSDoc type annotations** in TypeScript:

```typescript
// ✗ BAD - redundant
/**
 * @param {number} amountLitres
 * @return {boolean}
 */
brew(amountLitres: number): boolean { ... }

// ✓ GOOD
/**
 * @param amountLitres The amount to brew.
 */
brew(amountLitres: number): boolean { ... }
```

### Parameter Name Comments

Use when parameter meaning isn't clear:

```typescript
// ✓ GOOD
someFunction(obviousParam, /* shouldRender= */ true, /* name= */ 'hello');

// Consider refactoring to use an interface instead
interface Options {
  shouldRender: boolean;
  name: string;
}
someFunction(obviousParam, {shouldRender: true, name: 'hello'});
```

---

## Policies

### Consistency

- Follow what the file already does
- New files must use Google Style
- When reformatting, do it in a separate change

### Deprecation

Mark with `@deprecated` and provide clear migration path:

```typescript
/**
 * @deprecated Use newMethod() instead.
 */
oldMethod() { ... }
```

### Disallowed Features

**Do not use:**
- Wrapper objects for primitives (`new String()`, `new Boolean()`, `new Number()`)
- `const enum` (use plain `enum`)
- Debugger statements in production
- `with` keyword
- `eval` or `Function(...string)` constructor
- Non-standard features
- Modifying builtin objects

### Automatic Semicolon Insertion

**Do not rely on ASI.** Always use explicit semicolons.

### Toolchain

**TypeScript Compiler:**
- All code must pass type checking
- Do not use `@ts-ignore`, `@ts-expect-error`, or `@ts-nocheck`
- Exception: `@ts-expect-error` in unit tests (use sparingly)

---

## Summary

This guide provides comprehensive rules for writing TypeScript code that is:
- **Consistent** - Follows established patterns
- **Maintainable** - Easy to understand and modify
- **Type-safe** - Leverages TypeScript's type system
- **Scalable** - Works well in large codebases

For the complete official guide with all details and rationale, see:
https://google.github.io/styleguide/tsguide.html

---

**Last Updated:** Based on Google's official TypeScript Style Guide  
**License:** CC-By 3.0 (Google Inc.)
