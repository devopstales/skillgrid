# Google Markdown Style Guide

> Source: https://google.github.io/styleguide/docguide/style.html

## Golden Rules

1. **One sentence per line** — for easier diffs and editing
2. **ATX-style headers** — use `#` not underlines
3. **Fenced code blocks** — use ` ``` ` with language identifier
4. **Reference-style links** — for readability
5. **Consistent list markers** — `*` for unordered, `1.` for ordered

---

## 1. Headers

```markdown
<!-- CORRECT - ATX-style headers -->
# H1 Header
## H2 Header
### H3 Header

<!-- AVOID - Setext-style headers -->
H1 Header
=========

H2 Header
---------
```

---

## 2. Line Breaks

```markdown
<!-- CORRECT - one sentence per line -->
This is the first sentence.
This is the second sentence.
This is the third sentence.

<!-- AVOID - multiple sentences on one line -->
This is the first sentence. This is the second sentence. This is the third sentence.
```

---

## 3. Lists

```markdown
<!-- CORRECT - unordered lists with * -->
* First item
* Second item
* Third item

<!-- CORRECT - ordered lists -->
1. First item
1. Second item
1. Third item

<!-- CORRECT - nested lists -->
* Parent item
  * Child item
  * Another child
* Another parent
```

---

## 4. Code

```markdown
<!-- CORRECT - inline code -->
Use the `print()` function to output text.

<!-- CORRECT - fenced code blocks with language -->
```python
def hello():
    print("Hello, World!")
```

<!-- CORRECT - code block without language -->
```
Plain text code block
```

<!-- AVOID - indented code blocks -->
    def hello():
        print("Hello")
```

---

## 5. Links

```markdown
<!-- CORRECT - inline links -->
Visit [Google](https://www.google.com) for search.

<!-- CORRECT - reference-style links (preferred for readability) -->
Visit [Google][google-link] for search.
Check out the [style guide][style-guide].

[google-link]: https://www.google.com
[style-guide]: https://google.github.io/styleguide/

<!-- CORRECT - automatic links -->
<https://www.google.com>
```

---

## 6. Images

```markdown
<!-- CORRECT - inline image -->
![Alt text](image.png)

<!-- CORRECT - reference-style image -->
![Alt text][logo]

[logo]: image.png "Logo title"

<!-- CORRECT - image with link -->
[![Alt text](image.png)](https://www.google.com)
```

---

## 7. Emphasis

```markdown
<!-- CORRECT - italic -->
This is *italic* text.
This is _also italic_ text.

<!-- CORRECT - bold -->
This is **bold** text.
This is __also bold__ text.

<!-- CORRECT - bold and italic -->
This is ***bold and italic*** text.
```

---

## 8. Tables

```markdown
<!-- CORRECT - tables with alignment -->
| Name    | Age | City      |
|---------|----:|-----------|
| Alice   |  30 | New York  |
| Bob     |  25 | London    |
| Charlie |  35 | Tokyo     |

<!-- Left-aligned | Right-aligned | Center-aligned -->
| Left | Right | Center |
|:-----|------:|:------:|
| A    |     1 |   X    |
| B    |     2 |   Y    |
```

---

## 9. Blockquotes

```markdown
<!-- CORRECT - blockquotes -->
> This is a blockquote.
> It can span multiple lines.

<!-- CORRECT - nested blockquotes -->
> This is a blockquote.
>
> > This is a nested blockquote.
```

---

## 10. Horizontal Rules

```markdown
<!-- CORRECT - horizontal rule -->
---

<!-- ALSO CORRECT -->
***

<!-- ALSO CORRECT -->
___
```

---

## 11. Task Lists

```markdown
<!-- CORRECT - task lists (GitHub-flavored) -->
- [x] Completed task
- [ ] Incomplete task
- [ ] Another incomplete task
```

---

## 12. Escaping

```markdown
<!-- CORRECT - escape special characters -->
Use \* for literal asterisks.
Use \# for literal hash marks.
Use \` for literal backticks.
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Multiple sentences per line | One sentence per line |
| Setext headers | Use ATX-style `#` headers |
| Indented code blocks | Use fenced code blocks with ` ``` ` |
| No language in code blocks | Specify language: ` ```python ` |
| Inconsistent list markers | Use `*` for unordered, `1.` for ordered |
| Long inline links | Use reference-style links |

