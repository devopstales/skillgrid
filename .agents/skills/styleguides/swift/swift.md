# Google Swift Style Guide

> Source: https://google.github.io/styleguide/swift.html

## Golden Rules

1. **Follow Apple's Swift API Design Guidelines** — as baseline
2. **2-space indentation** — no tabs
3. **100-character line limit** — for readability
4. **Use `let` over `var`** — prefer immutability
5. **Explicit `self` only when required** — by compiler
6. **Use type inference** — when type is clear

---

## 1. Naming

| Element | Convention | Example |
|---|---|---|
| Types | UpperCamelCase | `UserService`, `NetworkManager` |
| Functions/Methods | lowerCamelCase | `fetchUser()`, `calculateTotal()` |
| Variables/Properties | lowerCamelCase | `userName`, `isActive` |
| Constants | lowerCamelCase | `maxRetries`, `defaultTimeout` |
| Enums | UpperCamelCase | `Status`, `NetworkError` |
| Enum cases | lowerCamelCase | `.active`, `.inactive` |

---

## 2. Types

```swift
// CORRECT - struct for value types
struct User {
    let id: Int
    let name: String
    var email: String
}

// CORRECT - class for reference types
class UserService {
    private let repository: UserRepository
    
    init(repository: UserRepository) {
        self.repository = repository
    }
    
    func fetchUser(id: Int) -> User? {
        return repository.find(id: id)
    }
}

// CORRECT - enum with associated values
enum Result<T> {
    case success(T)
    case failure(Error)
}
```

---

## 3. Properties

```swift
// CORRECT - computed property
var fullName: String {
    return "\(firstName) \(lastName)"
}

// CORRECT - property observer
var temperature: Double {
    didSet {
        if temperature > maxTemperature {
            triggerAlert()
        }
    }
}

// CORRECT - lazy property
lazy var expensiveResource: Resource = {
    return Resource()
}()
```

---

## 4. Functions

```swift
// CORRECT - function with parameters
func greet(person: String, from hometown: String) -> String {
    return "Hello \(person)! Glad you could visit from \(hometown)."
}

// CORRECT - function with default parameters
func connect(timeout: TimeInterval = 30) {
    // Implementation
}

// CORRECT - function with closure parameter
func performAsync(completion: @escaping (Result<Data>) -> Void) {
    // Implementation
}
```

---

## 5. Optionals

```swift
// CORRECT - optional binding
if let user = optionalUser {
    print(user.name)
}

// CORRECT - guard for early exit
guard let user = optionalUser else {
    return
}

// CORRECT - optional chaining
let length = user?.name?.count

// CORRECT - nil-coalescing
let name = user?.name ?? "Unknown"

// AVOID - force unwrapping
let name = user!.name  // AVOID unless absolutely certain
```

---

## 6. Error Handling

```swift
// CORRECT - throwing function
enum NetworkError: Error {
    case invalidURL
    case noData
    case decodingFailed
}

func fetchData(from url: String) throws -> Data {
    guard let url = URL(string: url) else {
        throw NetworkError.invalidURL
    }
    // Fetch data
    return data
}

// CORRECT - do-catch
do {
    let data = try fetchData(from: urlString)
    process(data)
} catch NetworkError.invalidURL {
    print("Invalid URL")
} catch {
    print("Error: \(error)")
}

// CORRECT - try? for optional result
let data = try? fetchData(from: urlString)
```

---

## 7. Closures

```swift
// CORRECT - trailing closure syntax
users.filter { $0.isActive }
    .map { $0.name }
    .sorted()

// CORRECT - explicit parameter names when needed
users.filter { user in
    user.age > 18 && user.isActive
}

// CORRECT - capture list
someMethod { [weak self] result in
    guard let self = self else { return }
    self.handleResult(result)
}
```

---

## 8. Control Flow

```swift
// CORRECT - if statement
if condition {
    doSomething()
} else {
    doSomethingElse()
}

// CORRECT - switch with pattern matching
switch result {
case .success(let data):
    process(data)
case .failure(let error):
    handle(error)
}

// CORRECT - for-in loop
for user in users {
    print(user.name)
}

// CORRECT - while loop
while condition {
    doSomething()
}
```

---

## 9. Extensions

```swift
// CORRECT - organize code with extensions
extension User {
    var displayName: String {
        return "\(firstName) \(lastName)"
    }
    
    func isAdult() -> Bool {
        return age >= 18
    }
}

// CORRECT - protocol conformance in extension
extension User: Codable {}

extension User: Equatable {
    static func == (lhs: User, rhs: User) -> Bool {
        return lhs.id == rhs.id
    }
}
```

---

## 10. Protocols

```swift
// CORRECT - protocol definition
protocol UserRepository {
    func find(id: Int) -> User?
    func save(_ user: User)
}

// CORRECT - protocol with associated type
protocol Container {
    associatedtype Item
    func add(_ item: Item)
    func get(at index: Int) -> Item?
}
```

---

## 11. Generics

```swift
// CORRECT - generic function
func swap<T>(_ a: inout T, _ b: inout T) {
    let temp = a
    a = b
    b = temp
}

// CORRECT - generic type
struct Stack<Element> {
    private var items: [Element] = []
    
    mutating func push(_ item: Element) {
        items.append(item)
    }
    
    mutating func pop() -> Element? {
        return items.popLast()
    }
}
```

---

## 12. Access Control

```swift
// CORRECT - use appropriate access levels
public class UserService {
    private let repository: UserRepository
    internal var cacheEnabled = true
    
    public init(repository: UserRepository) {
        self.repository = repository
    }
    
    public func fetchUser(id: Int) -> User? {
        return repository.find(id: id)
    }
    
    private func clearCache() {
        // Implementation
    }
}
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Using `var` everywhere | Prefer `let` for immutability |
| Force unwrapping `!` | Use optional binding or `guard` |
| Unnecessary `self` | Only use when required by compiler |
| Not using `weak` in closures | Use `[weak self]` to avoid retain cycles |
| Ignoring errors | Handle with `do-catch` or `try?` |
| Overusing classes | Prefer structs for value types |

