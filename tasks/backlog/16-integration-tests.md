# Task 16: Integration Tests

## Objective
Full round-trip integration tests using temp directories and mock keychain.

## Acceptance Criteria
- Test: add profile → list shows it → current shows it → verify files created
- Test: add two profiles → use to switch → verify credentials swapped
- Test: add → remove → list shows empty
- Test: add → rename → verify old name gone, new name present
- Test: doctor on healthy setup → all checks pass
- Test: doctor on broken setup (missing keychain) → reports failure
- All tests use t.TempDir() and keyring.MockInit()

## Dependencies
- All implementation tasks (06-14)
