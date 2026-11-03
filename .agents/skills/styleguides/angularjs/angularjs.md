# Google AngularJS Style Guide

> Source: https://google.github.io/styleguide/angularjs-google-style.html

## Golden Rules

1. **Use Closure's `goog.require` and `goog.provide`** — for dependency management
2. **Controllers are classes** — define on prototype
3. **Use 'controller as' syntax** — export controller to scope
4. **Directives for DOM manipulation** — keep controllers DOM-free
5. **Use `module.service` for services** — not factory or provider
6. **Reserve `$` for Angular/jQuery** — don't prefix your own identifiers

---

## 1. Module Definition

```javascript
// CORRECT - define module with goog.provide
goog.provide('myapp.users.UserController');
goog.provide('myapp.users.userService');

// CORRECT - module definition
myapp.users.module = angular.module('myapp.users', [
  'ngRoute',
  'ngResource'
]);
```

---

## 2. Controllers

```javascript
// CORRECT - controller as class
/**
 * User controller.
 * @param {!myapp.users.UserService} userService
 * @constructor
 * @ngInject
 * @export
 */
myapp.users.UserController = function(userService) {
  /** @private {!myapp.users.UserService} */
  this.userService_ = userService;
  
  /** @export {string} */
  this.userName = '';
};

/**
 * Loads user data.
 * @param {number} userId
 * @export
 */
myapp.users.UserController.prototype.loadUser = function(userId) {
  this.userService_.getUser(userId).then(function(user) {
    this.userName = user.name;
  }.bind(this));
};

// Register controller
myapp.users.module.controller(
    'UserController',
    myapp.users.UserController);
```

---

## 3. Controller As Syntax

```html
<!-- CORRECT - use 'controller as' -->
<div ng-controller="myapp.users.UserController as userCtrl">
  <h1>{{userCtrl.userName}}</h1>
  <button ng-click="userCtrl.loadUser(123)">Load User</button>
</div>
```

---

## 4. Services

```javascript
// CORRECT - service as class
/**
 * User service.
 * @param {!angular.$http} $http
 * @constructor
 * @ngInject
 */
myapp.users.UserService = function($http) {
  /** @private {!angular.$http} */
  this.http_ = $http;
};

/**
 * Gets user by ID.
 * @param {number} userId
 * @return {!angular.$q.Promise}
 */
myapp.users.UserService.prototype.getUser = function(userId) {
  return this.http_.get('/api/users/' + userId);
};

// Register service
myapp.users.module.service('userService', myapp.users.UserService);
```

---

## 5. Directives

```javascript
// CORRECT - directive as function returning DDO
goog.provide('myapp.directives.userCard');

/**
 * User card directive.
 * @return {angular.Directive}
 */
myapp.directives.userCard = function() {
  return {
    restrict: 'E',
    scope: {
      user: '='
    },
    templateUrl: 'templates/user-card.html',
    controller: 'UserCardController',
    controllerAs: 'ctrl',
    bindToController: true
  };
};

// Register directive
myapp.module.directive('userCard', myapp.directives.userCard);
```

---

## 6. Dependency Injection

```javascript
// CORRECT - use @ngInject annotation
/**
 * @param {!angular.$http} $http
 * @param {!myapp.UserService} userService
 * @constructor
 * @ngInject
 */
myapp.MyController = function($http, userService) {
  this.http_ = $http;
  this.userService_ = userService;
};
```

---

## 7. Naming Conventions

```javascript
// CORRECT - naming
myapp.users.UserController  // Controller class
myapp.users.UserService     // Service class
myapp.directives.userCard   // Directive function

// AVOID - don't use $ prefix
myapp.users.$UserService    // AVOID
this.$myProperty = value;   // AVOID
```

---

## 8. Scopes

```javascript
// CORRECT - use controller properties, not $scope
myapp.MyController = function() {
  /** @export {string} */
  this.message = 'Hello';
};

// AVOID - building up $scope object
myapp.MyController = function($scope) {
  $scope.message = 'Hello';  // AVOID
};
```

---

## 9. Promises

```javascript
// CORRECT - use promises
myapp.UserService.prototype.getUser = function(userId) {
  return this.http_.get('/api/users/' + userId)
      .then(function(response) {
        return response.data;
      })
      .catch(function(error) {
        console.error('Failed to get user:', error);
        throw error;
      });
};
```

---

## 10. Testing

```javascript
// CORRECT - Jasmine test
describe('UserController', function() {
  var ctrl;
  var mockUserService;
  
  beforeEach(module('myapp.users'));
  
  beforeEach(inject(function($controller) {
    mockUserService = {
      getUser: jasmine.createSpy('getUser')
    };
    
    ctrl = $controller('UserController', {
      userService: mockUserService
    });
  }));
  
  it('should load user', function() {
    ctrl.loadUser(123);
    expect(mockUserService.getUser).toHaveBeenCalledWith(123);
  });
});
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Using `$scope` directly | Use 'controller as' syntax |
| Using `$` prefix | Reserve for Angular/jQuery |
| DOM in controllers | Use directives for DOM manipulation |
| Using `factory` | Use `service` for classes |
| Missing `@ngInject` | Add annotation for DI |
| Not using Closure | Use `goog.provide`/`goog.require` |

