---
name: csharp
description: Google's official C# style guide. Covers naming conventions, formatting, LINQ, async/await, XML documentation, and .NET best practices.
---

# Google C# Style Guide

> Official Google C# coding standards for consistent, maintainable .NET code.

## Golden Rules

1. **Follow .NET naming conventions** — PascalCase for public members
2. **Use async/await** for asynchronous operations
3. **LINQ for collections** — prefer declarative over imperative
4. **XML documentation** for public APIs
5. **Use `var`** when type is obvious from right side
6. **Prefer expression-bodied members** when concise
7. **Use nullable reference types** (C# 8.0+)

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Classes/Interfaces | PascalCase | `UserService`, `IRepository` |
| Methods | PascalCase | `GetUserById` |
| Properties | PascalCase | `UserCount` |
| Local variables | camelCase | `userCount` |
| Constants | PascalCase | `MaxRetries` |
| Private fields | _camelCase | `_userId` |
| Parameters | camelCase | `userId` |
| Enums | PascalCase | `Direction` |

### Variables and Types

```csharp
// ✓ CORRECT - use var when type is obvious
var users = new List<User>();
var count = users.Count;

// ✓ CORRECT - explicit type when not obvious
IEnumerable<User> result = GetUsers();

// ✓ CORRECT - nullable reference types (C# 8+)
string? optionalName = null;
string requiredName = "Alice";
```

### Properties and Fields

```csharp
// ✓ CORRECT - auto-properties
public string Name { get; set; }
public int Count { get; private set; }

// ✓ CORRECT - private fields with underscore prefix
private readonly string _connectionString;
private int _retryCount;

// ✓ CORRECT - expression-bodied property
public string FullName => $"{FirstName} {LastName}";
```

### Methods and Expressions

```csharp
// ✓ CORRECT - expression-bodied method
public string GetDisplayName() => $"{FirstName} {LastName}";

// ✓ CORRECT - traditional method body
public async Task<User> GetUserAsync(int id)
{
    var user = await _repository.FindAsync(id);
    return user ?? throw new NotFoundException(id);
}
```

### Async/Await

```csharp
// ✓ CORRECT - async all the way
public async Task<IEnumerable<User>> GetActiveUsersAsync()
{
    return await _db.Users
        .Where(u => u.IsActive)
        .ToListAsync();
}

// ✓ CORRECT - ConfigureAwait in library code
var result = await SomeAsyncMethod().ConfigureAwait(false);

// ✗ INCORRECT - blocking on async code
var result = SomeAsyncMethod().Result;  // can deadlock
```

### LINQ

```csharp
// ✓ CORRECT - LINQ for queries
var activeUsers = users
    .Where(u => u.IsActive)
    .OrderBy(u => u.LastName)
    .Select(u => new UserDto(u.Id, u.Name))
    .ToList();

// ✓ CORRECT - query syntax for complex joins
var result = from u in users
             join r in roles on u.RoleId equals r.Id
             where u.IsActive
             select new { u.Name, r.Title };
```

### XML Documentation

```csharp
// ✓ CORRECT - XML docs for public API
/// <summary>
/// Retrieves a user by their unique identifier.
/// </summary>
/// <param name="id">The user's unique ID.</param>
/// <returns>The user, or null if not found.</returns>
/// <exception cref="ArgumentException">Thrown when id is negative.</exception>
public Task<User?> GetUserByIdAsync(int id)
{
    // ...
}
```

### Interfaces and Dependency Injection

```csharp
// ✓ CORRECT - depend on interfaces
public class UserService
{
    private readonly IUserRepository _repository;

    public UserService(IUserRepository repository)
    {
        _repository = repository;
    }
}
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Blocking on async (`.Result`, `.Wait()`) | Use `async`/`await` all the way |
| Missing XML docs on public API | Document with `///` XML comments |
| `public` fields | Use properties with get/set |
| Ignoring nullability | Enable `<Nullable>enable</Nullable>` |
| Imperative loops for collections | Use LINQ |
| Magic strings | Use `nameof()` or constants |
| Catching `Exception` broadly | Catch specific exception types |

## When to Use This Guide

- Writing new C# code
- Refactoring existing C#
- Code reviews
- Setting up `.editorconfig` rules
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/csharp
```

## Full Guide

See [csharp.md](csharp.md) for complete details, examples, and edge cases.
