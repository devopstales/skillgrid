# Google HTML/CSS Style Guide

> Source: https://google.github.io/styleguide/htmlcssguide.html

## Golden Rules

1. **Use HTTPS** — for all embedded resources
2. **2-space indentation** — no tabs
3. **Lowercase everything** — elements, attributes, selectors
4. **Valid HTML/CSS** — use validators
5. **Semantic HTML** — use elements for their intended purpose
6. **Separate concerns** — structure (HTML) from presentation (CSS)

---

## HTML

### 1. Document Type

```html
<!-- CORRECT - always use HTML5 doctype -->
<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Page Title</title>
  </head>
  <body>
    <h1>Hello World</h1>
  </body>
</html>
```

---

### 2. Semantic HTML

```html
<!-- CORRECT - use semantic elements -->
<header>
  <nav>
    <a href="/">Home</a>
  </nav>
</header>

<main>
  <article>
    <h1>Article Title</h1>
    <p>Article content...</p>
  </article>
</main>

<footer>
  <p>&copy; 2024 Company</p>
</footer>

<!-- AVOID - div soup -->
<div class="header">
  <div class="nav">
    <div class="link">Home</div>
  </div>
</div>
```

---

### 3. Attributes

```html
<!-- CORRECT - use double quotes, lowercase -->
<img src="logo.png" alt="Company Logo">
<a href="/about" class="nav-link">About</a>

<!-- CORRECT - omit type for CSS and JS -->
<link rel="stylesheet" href="style.css">
<script src="script.js"></script>

<!-- AVOID -->
<img src='logo.png' alt='Company Logo'>  <!-- single quotes -->
<link rel="stylesheet" href="style.css" type="text/css">  <!-- unnecessary type -->
```

---

### 4. Formatting

```html
<!-- CORRECT - new line for block elements -->
<ul>
  <li>Item 1</li>
  <li>Item 2</li>
  <li>Item 3</li>
</ul>

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Age</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Alice</td>
      <td>30</td>
    </tr>
  </tbody>
</table>
```

---

### 5. Accessibility

```html
<!-- CORRECT - provide alt text -->
<img src="chart.png" alt="Sales chart showing 20% growth">

<!-- CORRECT - use labels for form inputs -->
<label for="email">Email:</label>
<input type="email" id="email" name="email">

<!-- CORRECT - use semantic headings -->
<h1>Main Title</h1>
<h2>Section Title</h2>
<h3>Subsection Title</h3>
```

---

## CSS

### 1. Naming

```css
/* CORRECT - lowercase with hyphens */
.nav-link { }
.button-primary { }
.user-profile { }

/* AVOID */
.navLink { }  /* camelCase */
.nav_link { }  /* underscores */
.NAVLINK { }  /* uppercase */
```

---

### 2. Selectors

```css
/* CORRECT - use classes, not IDs */
.example { }
.error { }

/* AVOID - ID selectors */
#example { }  /* AVOID */

/* AVOID - type selectors with classes */
ul.example { }  /* AVOID */
div.error { }  /* AVOID */
```

---

### 3. Properties

```css
/* CORRECT - use shorthand */
.box {
  margin: 0 1em 2em;
  padding: 0;
  font: 100%/1.6 palatino, georgia, serif;
  border-top: 0;
}

/* AVOID - longhand when shorthand available */
.box {
  margin-top: 0;
  margin-right: 1em;
  margin-bottom: 2em;
  margin-left: 1em;
}
```

---

### 4. Units

```css
/* CORRECT - omit units for 0 */
.box {
  margin: 0;
  padding: 0;
}

/* CORRECT - include leading 0 */
.box {
  font-size: 0.8em;
  opacity: 0.5;
}

/* CORRECT - use 3-char hex when possible */
.box {
  color: #ebc;
  background: #fff;
}
```

---

### 5. Formatting

```css
/* CORRECT - one selector per line */
h1,
h2,
h3 {
  font-weight: normal;
  line-height: 1.2;
}

/* CORRECT - space after colon */
.box {
  color: #333;
  background: #fff;
}

/* CORRECT - space before opening brace */
.box {
  display: block;
}

/* CORRECT - semicolon after every declaration */
.box {
  width: 100%;
  height: 50px;
}
```

---

### 6. Declaration Order (Optional)

```css
/* CORRECT - alphabetical order */
.box {
  background: #fff;
  border: 1px solid #ddd;
  color: #333;
  display: block;
  font-size: 1em;
  margin: 1em;
  padding: 1em;
  width: 100%;
}
```

---

### 7. Avoid !important

```css
/* AVOID - !important breaks cascade */
.example {
  font-weight: bold !important;  /* AVOID */
}

/* CORRECT - use specificity */
.example {
  font-weight: bold;
}
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Using `<div>` for everything | Use semantic HTML5 elements |
| Missing alt text | Always provide meaningful alt text |
| ID selectors in CSS | Use class selectors |
| Inline styles | Use external stylesheets |
| `!important` overuse | Use proper specificity |
| Missing doctype | Always include `<!doctype html>` |
| Type attributes | Omit for CSS/JS (HTML5 default) |

