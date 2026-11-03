---
name: angularjs
description: Google's official AngularJS style guide. Covers controllers, services, directives, modules, dependency injection, and AngularJS 1.x best practices.
---

# Google AngularJS Style Guide

> Official Google AngularJS coding standards for consistent Angular 1.x applications.

**Note:** This guide covers AngularJS (Angular 1.x). For modern Angular (2+), see the Angular style guide.

## Golden Rules

1. **One component per file** — easier to maintain and test
2. **Use controllerAs syntax** — avoid `$scope` when possible
3. **Services for business logic** — keep controllers thin
4. **Explicit dependency injection** — array annotation or `$inject`
5. **Modular structure** — organize by feature, not type
6. **Use directives** — for DOM manipulation only
7. **Avoid `$rootScope`** — use services for shared state

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Modules | lowerCamelCase | `myApp`, `myApp.users` |
| Controllers | UpperCamelCase + Ctrl | `UserCtrl`, `HomeCtrl` |
| Services/Factories | UpperCamelCase | `UserService`, `AuthFactory` |
| Directives | lowerCamelCase | `myDirective`, `userCard` |
| Filters | lowerCamelCase | `dateFormat`, `currencyDisplay` |
| Files | feature.type.js | `user.controller.js` |

### Controllers

```javascript
// ✓ CORRECT - controllerAs syntax
angular.module('myApp')
  .controller('UserCtrl', UserCtrl);

UserCtrl.$inject = ['UserService', '$log'];

function UserCtrl(UserService, $log) {
  var vm = this;  // vm = viewModel

  vm.users = [];
  vm.loadUsers = loadUsers;

  activate();

  function activate() {
    loadUsers();
  }

  function loadUsers() {
    return UserService.getAll()
      .then(function(users) {
        vm.users = users;
        return vm.users;
      });
  }
}
```

### Services

```javascript
// ✓ CORRECT - service for business logic
angular.module('myApp')
  .factory('UserService', UserService);

UserService.$inject = ['$http', '$log'];

function UserService($http, $log) {
  var service = {
    getAll: getAll,
    getById: getById,
    create: create
  };

  return service;

  function getAll() {
    return $http.get('/api/users')
      .then(function(response) {
        return response.data;
      });
  }

  function getById(id) {
    return $http.get('/api/users/' + id)
      .then(function(response) {
        return response.data;
      });
  }

  function create(user) {
    return $http.post('/api/users', user);
  }
}
```

### Directives

```javascript
// ✓ CORRECT - directive for DOM manipulation
angular.module('myApp')
  .directive('userCard', userCard);

function userCard() {
  return {
    restrict: 'E',
    scope: {
      user: '=',
      onSelect: '&'
    },
    templateUrl: 'user-card.html',
    controller: 'UserCardCtrl',
    controllerAs: 'vm',
    bindToController: true
  };
}
```

### Dependency Injection

```javascript
// ✓ CORRECT - explicit $inject annotation (minification-safe)
MyCtrl.$inject = ['$scope', '$http', 'UserService'];

function MyCtrl($scope, $http, UserService) {
  // ...
}

// ✗ INCORRECT - inline array (verbose) or no annotation (breaks minification)
angular.module('myApp').controller('MyCtrl', ['$scope', function($scope) {}]);
```

### Modules

```javascript
// ✓ CORRECT - one module definition per file
// app.module.js
angular.module('myApp', ['ngRoute', 'myApp.users', 'myApp.auth']);

// users/users.module.js
angular.module('myApp.users', []);

// ✗ INCORRECT - defining everything in one file
angular.module('myApp', []).controller(...).service(...).directive(...);
```

### Templates

```html
<!-- ✓ CORRECT - controllerAs in template -->
<div ng-controller="UserCtrl as vm">
  <h1>{{ vm.title }}</h1>
  <ul>
    <li ng-repeat="user in vm.users track by user.id">
      {{ user.name }}
    </li>
  </ul>
  <button ng-click="vm.loadUsers()">Reload</button>
</div>
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Business logic in controllers | Move to services/factories |
| Using `$scope` directly | Use `controllerAs` with `vm` alias |
| Implicit DI (breaks minification) | Use `$inject` or array annotation |
| DOM manipulation in controllers | Use directives |
| Everything in one module | Organize by feature into sub-modules |
| Using `$rootScope` for shared state | Use a service instead |
| Watchers for every change | Use one-time binding `::` where possible |

## When to Use This Guide

- Maintaining AngularJS 1.x applications
- Code reviews for legacy Angular projects
- Onboarding new team members to AngularJS projects

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/angularjs
```

## Full Guide

See [angularjs.md](angularjs.md) for complete details, examples, and edge cases.
