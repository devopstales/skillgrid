---
name: swift
description: Google's official Swift style guide. Covers naming conventions, optionals, protocols, error handling, access control, formatting, and Swift best practices.
---

# Google Swift Style Guide

> Official Google Swift coding standards for consistent, maintainable iOS/macOS code.

## Golden Rules

1. **Clear, descriptive names** — readability at the call site
2. **Prefer `let` over `var`** — immutability by default
3. **Use optionals properly** — avoid force unwrapping (`!`)
4. **Protocol-oriented programming** — prefer protocols over inheritance
5. **`guard` for early returns** — reduce nesting with early exits
6. **Use type inference** — let Swift infer types when obvious
7. **SwiftLint for consistency** — automate style enforcement

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Types (class/struct/enum/protocol) | UpperCamelCase | `UserService` |
| Functions/Methods | lowerCamelCase | `getUserById` |
| Variables/Properties | lowerCamelCase | `userCount` |
| Constants | lowerCamelCase | `maxRetries` |
| Enums | UpperCamelCase | `Direction` |
| Enum cases | lowerCamelCase | `.up`, `.down`, `.left` |
| Type aliases | UpperCamelCase | `CompletionHandler` |

### Variables and Constants

```swift
// ✓ CORRECT - prefer let (immutable)
let userName = "Alice"
let maxRetries = 3

// ✓ CORRECT - var only when value changes
var retryCount = 0
retryCount += 1

// ✓ CORRECT - type inference (no need to repeat type)
let users: [User] = []        // OK: collection type is clear
let count = users.count       // inferred as Int
```

### Optionals

```swift
// ✓ CORRECT - if let for optional binding
if let user = getUser(id: 42) {
  print(user.name)
}

// ✓ CORRECT - guard for early return
func processUser(id: Int) {
  guard let user = getUser(id: id) else {
    print("User not found")
    return
  }
  // user is non-optional here
  display(user)
}

// ✓ CORRECT - nil coalescing
let name = user?.name ?? "Unknown"

// ✗ INCORRECT - force unwrap
let user = getUser(id: 42)!  // crashes if nil
```

### Structs and Classes

```swift
// ✓ CORRECT - prefer struct for value types
struct User {
  let id: Int
  var name: String
  var email: String
}

// ✓ CORRECT - class for reference semantics / inheritance
final class UserService {
  private let repository: UserRepository

  init(repository: UserRepository) {
    self.repository = repository
  }

  func getUser(id: Int) async throws -> User {
    return try await repository.find(id: id)
  }
}
```

### Protocols

```swift
// ✓ CORRECT - protocol for abstraction
protocol UserRepository {
  func find(id: Int) async throws -> User
  func save(_ user: User) async throws
}

// ✓ CORRECT - protocol extension for default behavior
extension UserRepository {
  func findAll() async throws -> [User] {
    return []
  }
}

// ✓ CORRECT - composing protocols
typealias UserStore = UserRepository & Codable
```

### Error Handling

```swift
// ✓ CORRECT - typed errors with enum
enum UserError: Error {
  case notFound(id: Int)
  case invalidData(String)
  case networkFailure(underlying: Error)
}

// ✓ CORRECT - throw and catch
func fetchUser(id: Int) throws -> User {
  guard let user = database[id] else {
    throw UserError.notFound(id: id)
  }
  return user
}

// ✓ CORRECT - do-catch
do {
  let user = try fetchUser(id: 42)
  display(user)
} catch UserError.notFound(let id) {
  print("User \(id) not found")
} catch {
  print("Unexpected error: \(error)")
}
```

### Async/Await

```swift
// ✓ CORRECT - async/await (Swift 5.5+)
func loadUsers() async throws -> [User] {
  let data = try await networkClient.get("/api/users")
  return try JSONDecoder().decode([User].self, from: data)
}

// ✓ CORRECT - task for concurrent work
Task {
  do {
    let users = try await loadUsers()
    await MainActor.run { self.users = users }
  } catch {
    await MainActor.run { self.error = error }
  }
}
```

### Access Control

```swift
// ✓ CORRECT - explicit access control
public struct User {
  public let id: Int
  public var name: String
  internal var metadata: [String: String]
  private var _internalState: Bool
}

// ✓ CORRECT - use final to prevent subclassing
public final class AuthService {
  // ...
}
```

### Formatting

```swift
// ✓ CORRECT - trailing closures
users.forEach { user in
  print(user.name)
}

let activeUsers = users.filter { $0.isActive }
let names = users.map { $0.name }

// ✓ CORRECT - 2-space indentation, opening brace on same line
func process() {
  if condition {
    doSomething()
  }
}
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Force unwrapping (`!`) | Use `if let`, `guard let`, or nil coalescing |
| `var` everywhere | Use `let` by default; `var` only when mutating |
| Class for everything | Prefer `struct` for value semantics |
| No error handling | Use `throws` / `do-catch` |
| Inheritance over protocols | Favor protocol composition |
| No access control | Explicitly mark `public`, `internal`, `private` |
| Implicit self in closures | Use `[weak self]` to avoid retain cycles |

## When to Use This Guide

- Writing new Swift code
- Refactoring existing Swift
- Code reviews
- Setting up SwiftLint rules
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/swift
```

## Full Guide

See [swift.md](swift.md) for complete details, examples, and edge cases.
