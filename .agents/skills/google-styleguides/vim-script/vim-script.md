# Google Vim Script Style Guide

> Source: https://google.github.io/styleguide/vimscriptguide.xml

## Golden Rules

1. **Use 2-space indentation** — for readability
2. **Prefix plugin functions** — with plugin name
3. **Use `abort` on functions** — for error handling
4. **Prefer `scriptencoding utf-8`** — at top of file
5. **Use `l:` for local variables** — explicit scoping
6. **Document all functions** — with comments

---

## 1. File Structure

```vim
" CORRECT - file header
" Plugin: MyPlugin
" Description: Does something useful
" Maintainer: Your Name <email@example.com>
" License: Apache 2.0

if exists('g:loaded_myplugin')
  finish
endif
let g:loaded_myplugin = 1

scriptencoding utf-8

" Plugin code here
```

---

## 2. Naming

| Element | Convention | Example |
|---|---|---|
| Functions | PrefixedCamelCase | `myplugin#DoSomething()` |
| Global vars | g:plugin_name | `g:myplugin_enabled` |
| Script vars | s:variable_name | `s:internal_state` |
| Local vars | l:variable_name | `l:temp_value` |
| Buffer vars | b:variable_name | `b:current_mode` |

```vim
" CORRECT - function naming
function! myplugin#ProcessBuffer() abort
  let l:lines = getline(1, '$')
  " Process lines
endfunction

" CORRECT - variable scoping
let g:myplugin_enabled = 1
let s:internal_counter = 0
```

---

## 3. Functions

```vim
" CORRECT - function with abort
function! myplugin#CalculateTotal(numbers) abort
  " Calculate sum of numbers
  let l:total = 0
  for l:num in a:numbers
    let l:total += l:num
  endfor
  return l:total
endfunction

" CORRECT - function with range
function! myplugin#ProcessRange() range abort
  for l:line_num in range(a:firstline, a:lastline)
    let l:line = getline(l:line_num)
    " Process line
  endfor
endfunction

" CORRECT - function with optional arguments
function! myplugin#Greet(...) abort
  let l:name = a:0 >= 1 ? a:1 : 'World'
  echo 'Hello, ' . l:name . '!'
endfunction
```

---

## 4. Variables

```vim
" CORRECT - explicit scoping
function! myplugin#Example() abort
  let l:local_var = 'local'
  let s:script_var = 'script'
  let g:global_var = 'global'
  
  " Use variables
  echo l:local_var
endfunction

" CORRECT - checking if variable exists
if exists('g:myplugin_config')
  let l:config = g:myplugin_config
else
  let l:config = {}
endif
```

---

## 5. Conditionals

```vim
" CORRECT - if statement
if condition
  " Do something
elseif other_condition
  " Do something else
else
  " Default action
endif

" CORRECT - checking for features
if has('python3')
  " Use Python 3
elseif has('python')
  " Use Python 2
else
  echoerr 'Python not available'
endif
```

---

## 6. Loops

```vim
" CORRECT - for loop
for l:item in l:items
  echo l:item
endfor

" CORRECT - while loop
let l:i = 0
while l:i < 10
  echo l:i
  let l:i += 1
endwhile

" CORRECT - iterating over range
for l:i in range(1, 10)
  echo l:i
endfor
```

---

## 7. Commands

```vim
" CORRECT - define custom command
command! -nargs=1 MyPluginGreet call myplugin#Greet(<f-args>)

" CORRECT - command with range
command! -range MyPluginProcess <line1>,<line2>call myplugin#ProcessRange()

" CORRECT - command with completion
command! -nargs=1 -complete=file MyPluginOpen call myplugin#Open(<f-args>)
```

---

## 8. Mappings

```vim
" CORRECT - normal mode mapping
nnoremap <silent> <Leader>mp :call myplugin#Process()<CR>

" CORRECT - visual mode mapping
vnoremap <silent> <Leader>mp :call myplugin#ProcessSelection()<CR>

" CORRECT - insert mode mapping
inoremap <silent> <C-Space> <C-R>=myplugin#Complete()<CR>

" CORRECT - buffer-local mapping
nnoremap <buffer> <silent> <Leader>mp :call myplugin#ProcessBuffer()<CR>
```

---

## 9. Autocommands

```vim
" CORRECT - autocommand group
augroup MyPlugin
  autocmd!
  autocmd BufRead,BufNewFile *.txt call myplugin#SetupTextFile()
  autocmd FileType python call myplugin#SetupPython()
augroup END
```

---

## 10. Error Handling

```vim
" CORRECT - try-catch
function! myplugin#SafeOperation() abort
  try
    " Risky operation
    call myplugin#RiskyFunction()
  catch /^Vim\%((\a\+)\)\=:E/
    echoerr 'Operation failed: ' . v:exception
  finally
    " Cleanup
    call myplugin#Cleanup()
  endtry
endfunction

" CORRECT - checking for errors
if !executable('git')
  echoerr 'Git not found'
  finish
endif
```

---

## 11. String Operations

```vim
" CORRECT - string concatenation
let l:message = 'Hello, ' . l:name . '!'

" CORRECT - string comparison
if l:str ==# 'exact'  " Case-sensitive
  echo 'Match'
endif

if l:str ==? 'case'  " Case-insensitive
  echo 'Match'
endif

" CORRECT - string functions
let l:upper = toupper(l:str)
let l:lower = tolower(l:str)
let l:trimmed = trim(l:str)
```

---

## 12. Lists and Dictionaries

```vim
" CORRECT - lists
let l:items = ['apple', 'banana', 'cherry']
let l:first = l:items[0]
call add(l:items, 'date')

" CORRECT - dictionaries
let l:config = {
      \ 'host': 'localhost',
      \ 'port': 8080,
      \ 'enabled': 1
      \ }
let l:host = l:config['host']
let l:port = get(l:config, 'port', 80)
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Missing `abort` | Always use `abort` on functions |
| No variable scoping | Use `l:`, `s:`, `g:` prefixes |
| Global namespace pollution | Prefix functions with plugin name |
| Missing guard clause | Check `g:loaded_plugin` at top |
| No autocommand group | Wrap autocommands in `augroup` |
| Case-insensitive comparison | Use `==#` or `==?` explicitly |

