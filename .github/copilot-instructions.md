# Copilot code review instructions — azure-extension-platform

This repo is a **shared Go library** consumed by multiple Azure VM extensions (e.g. VMApplication-Extension) on **Linux and Windows**. A bug or breaking change here ships to every downstream extension. Review with that blast radius in mind.

## Highest-priority concerns

1. **Public API compatibility.** Flag any change to exported identifiers in `vmextension/`, `pkg/**`, or `extensionlauncher/` (renamed/removed functions, changed signatures, struct field removals, changed `json` tags, changed error variables in `pkg/extensionerrors`, changed exit codes in `pkg/exithelper`). Ask whether downstream extensions were considered and whether the change should be additive instead.
2. **Cross-platform parity.** OS-specific code uses `_linux.go` / `_windows.go` file suffixes (e.g. `pkg/seqno`, `pkg/lockedfile`, `pkg/decrypt`, `pkg/utils`, `pkg/commandhandler`). If one platform's file changes, check whether the sibling needs the same change and that both still compile. CI builds and tests on both `ubuntu-latest` and `windows-latest` with `CGO_ENABLED=0`.
3. **Secrets handling.** Protected settings are decrypted in `pkg/settings` / `pkg/decrypt`. Never log, print, return in errors, or write to status/event files: `ProtectedSettings`, decrypted payloads, certificate material, SAS tokens, or full URLs with query strings. `settings.CleanUpSettings` must continue to scrub protected settings from `.settings` files.
4. **Guest Agent contract.** Status files (`pkg/status`) must stay compatible with the Guest Agent schema (`version`, `timestampUTC`, `status.operation`, `status.status` ∈ transitioning/error/success, `formattedMessage`, `substatus`) and keep the atomic write-to-temp-then-rename pattern. Sequence-number logic (`pkg/seqno`, `vmextension/enabledisable.go`) must not re-run an already-processed seqno when `requiresSeqNoChange` is set, and must not regress handling of missing mrseq/settings files.
5. **Process/exit behavior.** Use `exithelper.Exiter` instead of calling `os.Exit` directly so tests can intercept it. `extensionlauncher` starts the extension as an independent process; check for handle/file-descriptor leaks and correct transitioning-status writes.

## Go conventions in this repo

- Errors: wrap with `github.com/pkg/errors` (`errors.Wrap`/`Wrapf`) to add context; reuse sentinel errors from `pkg/extensionerrors` and compare with `==`/`errors.Is` rather than string matching. Don't silently ignore returned errors.
- Logging: use the `logging.ILogger` / `ExtensionLogger` passed in (`Info`/`Warn`/`Error`), not `fmt.Print*` or the std `log` package. Log messages must not contain secrets.
- File I/O: close files/readers (`defer f.Close()`), respect `lockedfile` for concurrent access, and avoid unbounded growth in log/event folders (see `logging.rotateLogFolder`).
- Testability: environment access goes through interfaces like `environmentmanager.IGetVMExtensionEnvironmentManager` and `seqno.ISequenceNumberRetriever`; new environment-dependent code should follow the same pattern so it can be mocked.
- Prefer `os`/`io` over deprecated `io/ioutil` in new code.

## Tests

- Tests use `testing` with `github.com/stretchr/testify` (`require` for preconditions, `assert` for checks). New behavior and bug fixes should come with tests, including platform-specific tests (`*_linux_test.go` / `*_windows_test.go`) when behavior differs by OS.
- Flag tests that have no assertions, depend on test execution order, write outside `t.TempDir()`, or leave environment variables (e.g. `ConfigSequenceNumber`) set after the test.

## Low-value feedback to avoid

- Don't nitpick formatting that `gofmt` handles, or comment style.
- Don't suggest large refactors of existing code that the PR didn't touch.
