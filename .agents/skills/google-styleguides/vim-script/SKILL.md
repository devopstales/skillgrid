---
name: vim-script
description: Google's official Vim script style guide. Covers plugin structure, naming conventions, variable scoping, error handling, portability, and Vimscript best practices.
---

# Google Vim Script Style Guide

> Official Google Vim script coding standards for consistent plugin development.

## Golden Rules

1. **Use `scriptencoding utf-8`** — declare encoding at top of every file
2. **Prefix plugin functions** — avoid namespace collisions with other plugins
3. **Use `abort` on functions** — fail fast on errors instead of silently continuing
4. **Check feature availability** — use `has('feature')` before using features
5. **Avoid global variables** — use script-local `s:` instead of `g:` internally
6. **Document all commands** — help users understand usage
7. **Test on multiple Vim versions** — ensure broad compatibility

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Plugin functions | `prefix#FunctionName` | `myplugin#GetUser` |
| Script-local functions | `s:FunctionName` | `s:InternalHelper` |
| Script-local variables | `s:variable_name` | `s:internal_state` |
| Local variables | `l:variable_name` | `l:temp_value` |
| Global variables | `g:plugin_name` | `g:myplugin_enabled` |
| Buffer variables | `b:variable_name` | `b:current_mode` |
| Autocommand groups | UpperCamelCase | `MyPlugin` |

### File Structure

```vim
" ✓ CORRECT - file header and guard
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

### Functions

```vim
" ✓ CORRECT - autoloaded function with abort
function! myplugin#ProcessBuffer() abort
  let l:lines = getline(1, '$')
  for l:line in l:lines
    call s:ProcessLine(l:line)
  endfor
endfunction

" ✓ CORRECT - script-local helper function
function! s:ProcessLine(line) abort
  if empty(a:line)
    return
  endif
  echo a:line
endfunction

" ✓ CORRECT - function with optional arguments
function! myplugin#Greet(...) abort
  let l:name = a:0 >= 1 ? a:1 : 'World'
  echo 'Hello, ' . l:name . '!'
endfunction
```

### Variables and Scoping

```vim
" ✓ CORRECT - explicit scoping
function! myplugin#Example() abort
  let l:local_var = 'local'
  let s:script_var = 'script'

  " Check before using global config
  if exists('g:myplugin_config')
    let l:config = g:myplugin_config
  else
    let l:config = {}
  endif

  echo l:local_var
endfunction
```

### Commands

```vim
" ✓ CORRECT - custom commands
command! -nargs=0 MyPluginRun call myplugin#Run()
command! -nargs=1 MyPluginGreet call myplugin#Greet(<f-args>)
command! -range MyPluginProcess <line1>,<line2>call myplugin#ProcessRange()
command! -nargs=1 -complete=file MyPluginOpen call myplugin#Open(<f-args>)
```

### Mappings

```vim
" ✓ CORRECT - non-recursive mappings with <silent>
nnoremap <silent> <Leader>mp :call myplugin#Process()<CR>
vnoremap <silent> <Leader>ms :call myplugin#ProcessSelection()<CR>
inoremap <silent> <C-Space> <C-R>=myplugin#Complete()<CR>

" ✓ CORRECT - buffer-local mapping
nnoremap <buffer> <silent> <Leader>mp :call myplugin#ProcessBuffer()<CR>
```

### Autocommands

```vim
" ✓ CORRECT - always use augroup to avoid duplicate autocommands
augroup MyPlugin
  autocmd!
  autocmd BufRead,BufNewFile *.txt call myplugin#SetupTextFile()
  autocmd FileType python call myplugin#SetupPython()
  autocmd VimLeave * call myplugin#Cleanup()
augroup END
```

### Error Handling

```vim
" ✓ CORRECT - try-catch for risky operations
function! myplugin#SafeOperation() abort
  try
    call myplugin#RiskyFunction()
  catch /^Vim\%((\\a\+)\)\=:E/
    echoerr 'Operation failed: ' . v:exception
  finally
    call myplugin#Cleanup()
  endtry
endfunction

" ✓ CORRECT - checking for required executables
if !executable('git')
  echoerr 'myplugin: git not found in PATH'
  finish
endif
```

### Feature Detection

```vim
" ✓ CORRECT - check features before using them
if has('python3')
  python3 import myplugin
elseif has('python')
  python import myplugin
else
  echoerr 'myplugin requires Python support'
  finish
endif

" ✓ CORRECT - check Vim version
if v:version < 800
  echoerr 'myplugin requires Vim 8.0+'
  finish
endif
```

### String Operations

```vim
" ✓ CORRECT - case-sensitive comparison
if l:str ==# 'exact'
  echo 'Case-sensitive match'
endif

" ✓ CORRECT - case-insensitive comparison
if l:str ==? 'case'
  echo 'Case-insensitive match'
endif

" ✓ CORRECT - string concatenation
let l:message = 'Hello, ' . l:name . '!'
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Missing `abort` on functions | Always use `function! Name() abort` |
| No variable scoping | Use `l:`, `s:`, `g:` prefixes explicitly |
| Global namespace pollution | Prefix all functions with plugin name |
| Missing `g:loaded_plugin` guard | Check at top to prevent double-loading |
| No `augroup` for autocommands | Always wrap in `augroup`/`autocmd!` |
| Case-ambiguous comparisons | Use `==#` (sensitive) or `==?` (insensitive) |
| Using `map` / `noremap` | Prefer `nnoremap`, `vnoremap`, etc. |

## When to Use This Guide

- Writing Vim plugins
- Maintaining Vim configuration files
- Code reviews for Vimscript
- Onboarding new contributors to Vim plugin projects

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/vim-script
```

## Full Guide

See [vim-script.md](vim-script.md) for complete details, examples, and edge cases.
