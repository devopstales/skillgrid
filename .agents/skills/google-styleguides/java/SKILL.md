---
name: java
description: Google's official Java style guide. Covers 2-space indentation, 100-char line limit, naming conventions, braces, imports, Javadoc, exception handling, lambdas, and streams. Enforces @Override annotations and specific imports.
---

# Google Java Style Guide

> Official Google Java coding standards for consistent, maintainable code.

## Golden Rules

1. **2-space indentation** — no tabs
2. **Column limit: 100 characters**
3. **Use `@Override`** whenever applicable
4. **No wildcard imports** — import specific types
5. **Braces required** even for single-statement blocks
6. **One top-level class per file**
7. **Prefer interfaces** for type definitions

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Packages | lowercase.dotted | `com.example.project` |
| Classes/Interfaces | UpperCamelCase | `UserService` |
| Methods | lowerCamelCase | `getUserById` |
| Variables | lowerCamelCase | `userCount` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRIES` |
| Type parameters | Single letter | `T`, `RequestT` |

### Classes

```java
// ✓ CORRECT
public class Animal {
    private final String name;

    public Animal(String name) {
        this.name = name;
    }

    public String speak() {
        return name + " makes a sound.";
    }
}
```

### Control Structures

```java
// ✓ CORRECT - braces always required
if (condition) {
    doSomething();
}

// ✗ INCORRECT - no braces
if (condition)
    doSomething();

// ✓ CORRECT - enhanced for loop
for (String item : items) {
    process(item);
}
```

### Exception Handling

```java
// ✓ CORRECT
try {
    processData(input);
} catch (IOException e) {
    logger.error("IO error: {}", e.getMessage());
    throw new ServiceException("Failed", e);
}

// ✓ CORRECT - try-with-resources
try (InputStream in = new FileInputStream(file)) {
    return IOUtils.toByteArray(in);
}

// ✗ INCORRECT
try {
    ...
} catch (Exception e) {    // too broad
    e.printStackTrace();   // don't use printStackTrace
}
```

### Lambdas and Streams

```java
// ✓ CORRECT
List<String> names = users.stream()
    .filter(u -> u.isActive())
    .map(User::getName)
    .sorted()
    .collect(Collectors.toList());

// ✓ CORRECT - method references
users.forEach(System.out::println);
```

### Javadoc

```java
/**
 * Finds a user by their unique identifier.
 *
 * @param userId the user's unique ID
 * @return an Optional containing the user, or empty if not found
 */
public Optional<User> findUserById(long userId) { ... }
```

### Imports

```java
// ✓ CORRECT
import java.util.List;
import java.util.Optional;

// ✗ INCORRECT
import java.util.*;  // no wildcard imports
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Wildcard imports | Specific imports only |
| Omitting braces | Always use braces |
| Missing @Override | Always annotate overridden methods |
| e.printStackTrace() | Use a proper logger |
| Catching Exception broadly | Catch specific exceptions |
| 4-space indentation | Use 2-space indentation |

## When to Use This Guide

- Writing new Java code
- Refactoring existing Java
- Code reviews
- Setting up Checkstyle rules
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/java
```

## Full Guide

See [java.md](java.md) for complete details, examples, and edge cases.
