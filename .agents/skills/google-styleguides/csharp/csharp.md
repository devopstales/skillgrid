# Google C# Style Guide

> Source: https://google.github.io/styleguide/csharp.html

## Golden Rules

1. **Follow Microsoft C# Coding Conventions** — as baseline
2. **PascalCase for public members** — methods, properties, classes
3. **camelCase for private fields** — with underscore prefix `_fieldName`
4. **Use `var` judiciously** — when type is obvious
5. **Prefer expression-bodied members** — for simple methods
6. **Use nullable reference types** — enable in C# 8.0+

---

## 1. Naming

| Element | Convention | Example |
|---|---|---|
| Classes/Interfaces | PascalCase | `UserService`, `IRepository` |
| Methods | PascalCase | `GetUserById()` |
| Properties | PascalCase | `FirstName` |
| Public fields | PascalCase | `MaxValue` |
| Private fields | _camelCase | `_userName` |
| Local variables | camelCase | `userCount` |
| Parameters | camelCase | `userId` |
| Constants | PascalCase | `MaxRetries` |

---

## 2. Classes and Interfaces

```csharp
// CORRECT
public class UserService
{
    private readonly IUserRepository _repository;
    private int _userCount;

    public UserService(IUserRepository repository)
    {
        _repository = repository;
    }

    public User GetUserById(int userId)
    {
        return _repository.FindById(userId);
    }
}

// CORRECT - interface naming
public interface IUserRepository
{
    User FindById(int id);
    void Save(User user);
}
```

---

## 3. Properties

```csharp
// CORRECT - auto-properties
public string FirstName { get; set; }
public int Age { get; private set; }
public bool IsActive { get; }

// CORRECT - expression-bodied property
public string FullName => $"{FirstName} {LastName}";

// CORRECT - property with backing field
private string _email;
public string Email
{
    get => _email;
    set => _email = value?.Trim();
}
```

---

## 4. Methods

```csharp
// CORRECT - expression-bodied method
public int Add(int a, int b) => a + b;

// CORRECT - regular method
public void ProcessUser(User user)
{
    if (user == null)
        throw new ArgumentNullException(nameof(user));
        
    _repository.Save(user);
}

// CORRECT - async method
public async Task<User> GetUserAsync(int userId)
{
    return await _repository.FindByIdAsync(userId);
}
```

---

## 5. Control Flow

```csharp
// CORRECT - braces on new line
if (condition)
{
    DoSomething();
}
else
{
    DoSomethingElse();
}

// CORRECT - switch expression (C# 8.0+)
var result = status switch
{
    Status.Active => "Active",
    Status.Inactive => "Inactive",
    _ => "Unknown"
};

// CORRECT - pattern matching
if (obj is User user)
{
    Console.WriteLine(user.Name);
}
```

---

## 6. LINQ

```csharp
// CORRECT - query syntax
var activeUsers = from user in users
                  where user.IsActive
                  orderby user.Name
                  select user;

// CORRECT - method syntax
var activeUsers = users
    .Where(u => u.IsActive)
    .OrderBy(u => u.Name)
    .ToList();
```

---

## 7. Null Handling

```csharp
// CORRECT - null-conditional operator
var length = user?.Name?.Length ?? 0;

// CORRECT - null-coalescing operator
var name = user.Name ?? "Unknown";

// CORRECT - nullable reference types (C# 8.0+)
public class User
{
    public string Name { get; set; } = string.Empty;  // Non-nullable
    public string? MiddleName { get; set; }  // Nullable
}
```

---

## 8. Exception Handling

```csharp
// CORRECT
try
{
    ProcessData();
}
catch (ArgumentException ex)
{
    _logger.LogError(ex, "Invalid argument");
    throw;
}
catch (Exception ex)
{
    _logger.LogError(ex, "Unexpected error");
    throw new ApplicationException("Processing failed", ex);
}
finally
{
    Cleanup();
}
```

---

## 9. Using Statements

```csharp
// CORRECT - using declaration (C# 8.0+)
using var stream = File.OpenRead("file.txt");
// stream is disposed at end of scope

// CORRECT - traditional using
using (var connection = new SqlConnection(connectionString))
{
    connection.Open();
    // Use connection
}
```

---

## 10. Modern C# Features

```csharp
// CORRECT - record types (C# 9.0+)
public record User(int Id, string Name);

// CORRECT - init-only properties (C# 9.0+)
public class User
{
    public int Id { get; init; }
    public string Name { get; init; }
}

// CORRECT - target-typed new (C# 9.0+)
User user = new(1, "Alice");
List<string> names = new();
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Public fields | Use properties instead |
| Hungarian notation | Use meaningful names without type prefixes |
| Catching `Exception` | Catch specific exceptions |
| Not using `async`/`await` | Use for I/O-bound operations |
| Ignoring nullable warnings | Enable and fix nullable reference types |
| Not disposing resources | Use `using` statements |

