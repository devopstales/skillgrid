# Verification

Verify work before claiming it is done — evidence before assertions:

- Run the relevant tests/build/lint and confirm the output passes.
- For CLI tools, run the command the user would run and check exit code + output.
- Prefer no-op confirmation: a change is done only when its success criterion (tests, command output, validation) has been observed.
