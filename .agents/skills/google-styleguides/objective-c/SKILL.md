---
name: objective-c
description: Google's official Objective-C style guide. Covers naming conventions, formatting, memory management, properties, protocols, categories, and Objective-C best practices.
---

# Google Objective-C Style Guide

> Official Google Objective-C coding standards for consistent iOS/macOS code.

## Golden Rules

1. **Use ARC** — Automatic Reference Counting (no manual retain/release)
2. **Descriptive naming** — clarity over brevity; method names read like sentences
3. **Prefix class names** — avoid namespace collisions (2-3 letter prefix)
4. **Use properties** — prefer `@property` over direct instance variables
5. **Nullability annotations** — mark `nullable`/`nonnull` on APIs
6. **Use modern Objective-C** — literals, subscripting, generics
7. **Follow Apple conventions** — consistency with Cocoa frameworks

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Classes | PrefixUpperCamelCase | `GTMUserService` |
| Protocols | PrefixUpperCamelCase | `GTMUserServiceDelegate` |
| Methods | lowerCamelCase | `- (void)getUserById:(NSInteger)userId` |
| Properties | lowerCamelCase | `userCount`, `firstName` |
| Local variables | lowerCamelCase | `localUser` |
| Constants | kPrefixUpperCamelCase | `kGTMMaxRetries` |
| Enums | PrefixUpperCamelCase | `GTMDirection` |
| Enum values | PrefixUpperCamelCase | `GTMDirectionNorth` |

### Class Interface

```objc
// ✓ CORRECT - class interface with nullability and generics
@interface GTMUserService : NSObject

@property(nonatomic, readonly) NSInteger userCount;
@property(nonatomic, copy, nullable) NSString *currentUserName;

- (nullable GTMUser *)userWithId:(NSInteger)userId;
- (void)fetchUsersWithCompletion:(void (^)(NSArray<GTMUser *> *_Nullable users,
                                          NSError *_Nullable error))completion;
@end
```

### Properties

```objc
// ✓ CORRECT - property declarations
@property(nonatomic, strong) NSMutableArray<GTMUser *> *users;
@property(nonatomic, copy) NSString *title;
@property(nonatomic, assign) BOOL isLoading;
@property(nonatomic, weak) id<GTMUserDelegate> delegate;
@property(nonatomic, readonly) NSInteger count;

// Attribute order: nonatomic/atomic, memory (strong/weak/copy/assign),
// then access (readonly/readwrite)
```

### Methods

```objc
// ✓ CORRECT - descriptive method names (reads like a sentence)
- (nullable GTMUser *)userWithFirstName:(NSString *)firstName
                               lastName:(NSString *)lastName;

// ✓ CORRECT - instance method
- (void)setUser:(GTMUser *)user forKey:(NSString *)key;

// ✓ CORRECT - class method (factory)
+ (instancetype)serviceWithBaseURL:(NSURL *)baseURL;

// ✗ INCORRECT - too abbreviated
- (GTMUser *)usr:(NSString *)fn ln:(NSString *)ln;
```

### Modern Objective-C Literals

```objc
// ✓ CORRECT - use modern literals
NSArray *colors = @[@"red", @"green", @"blue"];
NSDictionary *config = @{@"host": @"localhost", @"port": @8080};
NSNumber *count = @42;
NSNumber *enabled = @YES;

// ✓ CORRECT - subscripting
NSString *first = colors[0];
NSString *host = config[@"host"];

// ✗ INCORRECT - old style
NSArray *colors = [NSArray arrayWithObjects:@"red", @"green", nil];
```

### Protocols and Delegates

```objc
// ✓ CORRECT - optional and required protocol methods
@protocol GTMUserServiceDelegate <NSObject>

@required
- (void)userService:(GTMUserService *)service
    didLoadUser:(GTMUser *)user;

@optional
- (void)userService:(GTMUserService *)service
    didFailWithError:(NSError *)error;

@end
```

### Memory Management with ARC

```objc
// ✓ CORRECT - ARC handles retain/release
@implementation GTMUserService {
  NSMutableArray<GTMUser *> *_users;  // strong by default
}

- (instancetype)init {
  self = [super init];
  if (self) {
    _users = [[NSMutableArray alloc] init];
  }
  return self;
}
@end

// ✓ CORRECT - weak references to avoid cycles
__weak typeof(self) weakSelf = self;
[self fetchData:^{
  __strong typeof(weakSelf) strongSelf = weakSelf;
  if (!strongSelf) return;
  [strongSelf processResults];
}];
```

### Nullability Annotations

```objc
// ✓ CORRECT - annotate all APIs
NS_ASSUME_NONNULL_BEGIN

@interface GTMUser : NSObject

@property(nonatomic, copy) NSString *name;
@property(nonatomic, copy, nullable) NSString *email;

- (nullable GTMUser *)userWithId:(NSInteger)userId;

@end

NS_ASSUME_NONNULL_END
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Short/abbreviated names | Use descriptive, sentence-like names |
| No class prefix | Use 2-3 letter prefix to avoid collisions |
| Direct ivar access | Use `@property` and synthesized accessors |
| Missing nullability annotations | Use `NS_ASSUME_NONNULL_BEGIN/END` |
| Manual retain/release | Use ARC |
| Old collection syntax | Use modern literals `@[]`, `@{}` |
| Strong delegate references | Use `weak` for delegates |

## When to Use This Guide

- Writing new Objective-C code
- Maintaining legacy Objective-C iOS/macOS apps
- Code reviews
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/objective-c
```

## Full Guide

See [objective-c.md](objective-c.md) for complete details, examples, and edge cases.
