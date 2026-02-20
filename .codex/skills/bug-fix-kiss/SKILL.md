---
name: bug-fix-kiss
description: "Diagnose and fix software bugs with a strict root-cause-first workflow and KISS implementation strategy. Use when the user asks to fix a bug, debug, 排查并修复问题, 修 bug, 处理回归, or any request that requires safe bug resolution without speculative hacks. Enforce no reward-hack special casing, no fallback behavior without explicit user approval, and mandatory test verification to avoid breaking other modules."
---

# Bug Fix KISS

## Workflow

1. Reproduce the bug and define expected behavior.
2. Identify the real root cause before editing code.
3. Implement the smallest correct fix that follows KISS.
4. Run tests to confirm the fix and protect existing behavior.
5. Report root cause, fix scope, and verification results.

## Execution Rules

- Always investigate root cause first. Do not patch symptoms blindly.
- Keep changes minimal and local. Prefer the simplest valid design.
- Reject reward-hack style special-case logic that only satisfies one visible case.
- Do not add fallback logic unless the user explicitly approves it.
- Preserve existing contracts unless the user requests behavior changes.

## Root Cause Process

1. Collect failure evidence from logs, stack traces, and failing tests.
2. Trace backward to the first incorrect state or assumption.
3. Confirm causality with a minimal repro or focused check.
4. Name the cause explicitly before code edits.

## Fix Strategy (keep it simple and stupid - KISS)

1. Change the smallest number of files and lines needed.
2. Prefer direct, readable logic over abstraction or premature optimization.
3. Remove dead branches or brittle conditionals introduced by prior hacks.
4. Avoid unrelated refactors while fixing the bug.

## Verification Requirements

1. Run tests that cover the changed code path.
2. Run the relevant broader suite to ensure no regressions in neighboring modules.
3. Treat unresolved failures as blockers; do not claim completion without test evidence.
4. If test infrastructure is unavailable or broken, report the blocker clearly and request user direction.

## Optional Test Creation (Only On User Request)

When and only when the user asks for a corresponding test after the fix:

1. Add a focused regression test for the root cause.
2. Add a brief comment in the test describing the original bug scenario.
3. Keep the comment factual: trigger condition, incorrect behavior, expected behavior.
4. Keep the test deterministic and minimal.

## Output Template

Use this structure in final responses:

1. Root cause
2. Fix applied (KISS rationale)
3. Tests run and results
4. Remaining risks or blockers (if any)
