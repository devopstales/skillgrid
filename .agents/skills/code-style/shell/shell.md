# Google Shell Style Guide

> Source: https://google.github.io/styleguide/shellguide.html

## Golden Rules

1. **Use Bash** — `#!/bin/bash` for all shell scripts
2. **2-space indentation** — no tabs
3. **80-character line limit** — for readability
4. **Quote variables** — always use `"${var}"`
5. **Check return values** — never ignore errors
6. **Use ShellCheck** — lint your scripts
7. **Keep scripts under 100 lines** — or rewrite in another language

---

## 1. File Structure

```bash
#!/bin/bash
#
# Brief description of script purpose.
# Detailed usage information if needed.

set -euo pipefail  # Exit on error, undefined vars, pipe failures

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly CONST_VALUE="constant"

main() {
  # Main script logic here
  do_something "$@"
}

do_something() {
  local arg="$1"
  echo "Processing: ${arg}"
}

main "$@"
```

---

## 2. Naming

| Element | Convention | Example |
|---|---|---|
| Files | snake_case.sh | `backup_database.sh` |
| Functions | snake_case | `do_something()` |
| Variables | snake_case | `file_name` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRIES` |
| Environment vars | UPPER_SNAKE_CASE | `PATH` |

---

## 3. Functions

```bash
# CORRECT - function with comments
#######################################
# Cleanup files from backup directory.
# Globals:
#   BACKUP_DIR
# Arguments:
#   None
# Returns:
#   0 on success, 1 on error
#######################################
cleanup() {
  local file
  for file in "${BACKUP_DIR}"/*; do
    rm "${file}" || return 1
  done
  return 0
}

# CORRECT - function call
cleanup || {
  echo "Cleanup failed" >&2
  exit 1
}
```

---

## 4. Variables

```bash
# CORRECT - always quote variables
echo "${my_var}"
echo "${1}"
echo "${file_name}"

# CORRECT - use local in functions
my_function() {
  local local_var="value"
  echo "${local_var}"
}

# CORRECT - readonly for constants
readonly MAX_RETRIES=3
readonly CONFIG_FILE="/etc/myapp.conf"

# AVOID - unquoted variables
echo $my_var  # AVOID
echo $1  # AVOID
```

---

## 5. Conditionals

```bash
# CORRECT - use [[ ]] for tests
if [[ -f "${file}" ]]; then
  echo "File exists"
fi

if [[ "${var}" == "value" ]]; then
  do_something
fi

if [[ -z "${var}" ]]; then
  echo "Variable is empty"
fi

# CORRECT - numeric comparisons
if (( count > 10 )); then
  echo "Count is greater than 10"
fi

# AVOID - use [ ] (old test command)
if [ -f "${file}" ]; then  # Use [[ ]] instead
  echo "File exists"
fi
```

---

## 6. Loops

```bash
# CORRECT - iterate over array
for item in "${array[@]}"; do
  process "${item}"
done

# CORRECT - C-style for loop
for (( i = 0; i < 10; i++ )); do
  echo "${i}"
done

# CORRECT - while loop
while read -r line; do
  echo "Line: ${line}"
done < "${file}"

# CORRECT - process substitution (not pipe to while)
while read -r line; do
  echo "${line}"
done < <(command)
```

---

## 7. Error Handling

```bash
# CORRECT - check return values
if ! command; then
  echo "Command failed" >&2
  exit 1
fi

# CORRECT - use || for error handling
command || {
  echo "Command failed" >&2
  exit 1
}

# CORRECT - check $?
command
if (( $? != 0 )); then
  echo "Command failed" >&2
  exit 1
fi

# CORRECT - set options for safety
set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure
```

---

## 8. Command Substitution

```bash
# CORRECT - use $() not backticks
result="$(command)"
files="$(ls -1)"

# AVOID - backticks
result=`command`  # AVOID
```

---

## 9. Arrays

```bash
# CORRECT - declare and use arrays
declare -a files
files=("file1.txt" "file2.txt" "file3.txt")

# CORRECT - append to array
files+=("file4.txt")

# CORRECT - iterate over array
for file in "${files[@]}"; do
  echo "${file}"
done

# CORRECT - array length
echo "Array has ${#files[@]} elements"
```

---

## 10. Pipes and Redirection

```bash
# CORRECT - pipe to while with process substitution
while read -r line; do
  process "${line}"
done < <(command)

# CORRECT - redirect stderr
command 2>&1 | tee log.txt

# CORRECT - redirect to file
echo "output" > file.txt
echo "append" >> file.txt

# CORRECT - here document
cat <<EOF
Line 1
Line 2
EOF
```

---

## 11. Case Statements

```bash
# CORRECT - case statement
case "${option}" in
  start)
    start_service
    ;;
  stop)
    stop_service
    ;;
  restart)
    stop_service
    start_service
    ;;
  *)
    echo "Unknown option: ${option}" >&2
    exit 1
    ;;
esac
```

---

## 12. Best Practices

```bash
# CORRECT - use set for safety
set -euo pipefail

# CORRECT - use readonly for constants
readonly CONFIG_DIR="/etc/myapp"

# CORRECT - use local for function variables
my_func() {
  local temp_file
  temp_file="$(mktemp)"
  # Use temp_file
}

# CORRECT - use shellcheck
# shellcheck disable=SC2034  # Unused variable
unused_var="value"

# CORRECT - use meaningful variable names
user_count=10  # GOOD
uc=10  # AVOID
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Unquoted variables | Always quote: `"${var}"` |
| Using `[ ]` | Use `[[ ]]` instead |
| Backticks | Use `$()` for command substitution |
| Ignoring errors | Check return values with `if !` or `||` |
| Global variables | Use `local` in functions |
| Pipe to while | Use process substitution: `< <(cmd)` |
| Missing `set -e` | Add safety options at top of script |

