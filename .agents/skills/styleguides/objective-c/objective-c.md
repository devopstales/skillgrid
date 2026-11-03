# Google Objective-C Style Guide

> Source: https://google.github.io/styleguide/objcguide.html

## Golden Rules

1. **Follow Apple's Cocoa Coding Guidelines** — this guide extends them
2. **Use descriptive names** — clarity over brevity
3. **2-space indentation** — no tabs
4. **Prefix classes with 3+ characters** — avoid naming collisions
5. **Use ARC** — Automatic Reference Counting for memory management
6. **Document all public APIs** — with clear comments

---

## 1. Naming

### Classes and Protocols

```objc
// CORRECT - 3+ character prefix
@interface GTMExampleClass : NSObject
@end

@protocol GTMExampleDelegate <NSObject>
@end

// AVOID - no prefix or too short
@interface ExampleClass : NSObject  // AVOID
@end
```

### Methods

```objc
// CORRECT - descriptive, reads like a sentence
- (void)addTarget:(id)target action:(SEL)action;
- (CGPoint)convertPoint:(CGPoint)point fromView:(UIView *)view;

// CORRECT - getter without 'get' prefix
- (NSString *)title;
- (BOOL)isEnabled;

// AVOID
- (NSString *)getTitle;  // AVOID 'get' prefix
```

### Variables

```objc
// CORRECT - instance variables with underscore
@implementation MyClass {
  NSString *_instanceVariable;
}

// CORRECT - local variables
NSString *localVariable;
int loopCounter;

// CORRECT - constants
static const NSTimeInterval kAnimationDuration = 0.3;
```

---

## 2. File Structure

```objc
// MyClass.h
#import <Foundation/Foundation.h>

@class OtherClass;

/** Brief description of MyClass. */
@interface MyClass : NSObject

/** The main title. */
@property(nonatomic, copy) NSString *title;

/**
 * Initializes with a title.
 * @param title The title to use.
 */
- (instancetype)initWithTitle:(NSString *)title NS_DESIGNATED_INITIALIZER;

@end
```

---

## 3. Properties

```objc
// CORRECT - property declarations
@property(nonatomic, copy) NSString *name;
@property(nonatomic, strong) UIView *contentView;
@property(nonatomic, weak) id<MyDelegate> delegate;
@property(nonatomic, assign) NSInteger count;
@property(nonatomic, readonly) BOOL isValid;

// CORRECT - use copy for NSString, NSArray, etc.
@property(nonatomic, copy) NSArray<NSString *> *items;
```

---

## 4. Methods

```objc
// CORRECT - method implementation
- (instancetype)initWithTitle:(NSString *)title {
  self = [super init];
  if (self) {
    _title = [title copy];
  }
  return self;
}

// CORRECT - method with multiple parameters
- (void)doSomethingWithString:(NSString *)string
                       number:(NSInteger)number
                        error:(NSError **)error {
  // Implementation
}
```

---

## 5. Control Flow

```objc
// CORRECT - braces on same line
if (condition) {
  DoSomething();
} else {
  DoSomethingElse();
}

// CORRECT - for loops
for (NSInteger i = 0; i < count; i++) {
  Process(i);
}

// CORRECT - fast enumeration
for (NSString *item in array) {
  Process(item);
}

// CORRECT - switch statements
switch (value) {
  case 1:
    DoSomething();
    break;
  case 2:
    DoSomethingElse();
    break;
  default:
    break;
}
```

---

## 6. Blocks

```objc
// CORRECT - block as parameter
- (void)doAsyncWorkWithCompletion:(void (^)(NSError *error))completion {
  dispatch_async(queue, ^{
    // Work
    completion(nil);
  });
}

// CORRECT - block variable
void (^myBlock)(NSString *) = ^(NSString *input) {
  NSLog(@"%@", input);
};
```

---

## 7. Categories

```objc
// CORRECT - category naming
@interface NSString (GTMStringUtils)
- (NSString *)gtm_reversedString;
@end

// CORRECT - prefix methods to avoid collisions
@implementation NSString (GTMStringUtils)
- (NSString *)gtm_reversedString {
  // Implementation
}
@end
```

---

## 8. Protocols

```objc
// CORRECT - protocol definition
@protocol GTMDataSource <NSObject>

@required
- (NSInteger)numberOfItems;
- (id)itemAtIndex:(NSInteger)index;

@optional
- (NSString *)titleForItemAtIndex:(NSInteger)index;

@end
```

---

## 9. Comments

```objc
/**
 * A class representing a user profile.
 * Use this class to manage user data and preferences.
 */
@interface GTMUserProfile : NSObject

/**
 * The user's display name.
 * This is shown in the UI and can be edited by the user.
 */
@property(nonatomic, copy) NSString *displayName;

/**
 * Saves the profile to disk.
 * @param error On failure, contains an error object.
 * @return YES if successful, NO otherwise.
 */
- (BOOL)saveWithError:(NSError **)error;

@end
```

---

## 10. Modern Objective-C

```objc
// CORRECT - use literals
NSArray *array = @[@"one", @"two", @"three"];
NSDictionary *dict = @{@"key": @"value"};
NSNumber *number = @42;

// CORRECT - use subscripting
NSString *item = array[0];
NSString *value = dict[@"key"];

// CORRECT - use generics
NSArray<NSString *> *strings;
NSDictionary<NSString *, NSNumber *> *mapping;
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| No class prefix | Use 3+ character prefix (e.g., GTM) |
| `get` in getter names | Omit `get` prefix |
| Not using ARC | Use ARC for memory management |
| Retaining delegates | Use `weak` for delegates |
| Not copying NSString | Use `copy` for string properties |
| Missing nullability | Add `nullable`/`nonnull` annotations |

