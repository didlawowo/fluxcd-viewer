#!/usr/bin/env python3
"""Label-driven mutation testing policy shared by GitHub and Forgejo."""

from __future__ import annotations

import argparse
import json
import os
import re
from pathlib import Path
from typing import Callable
from urllib.request import Request, urlopen

MUTATION_REQUIRED_LABELS = ("complexity:high", "priority:high")
COMMENT_MARKER = "<!-- github-manager:mutation-policy:v1 -->"

_LINKED_ISSUE_RE = re.compile(
    r"(?i)\b(?:close[sd]?|fix(?:es|ed)?|resolve[sd]?|refs?)\s*:?\s*#(\d+)\b"
)

MUTATION_GUIDANCE = f"""{COMMENT_MARKER}
### Mutation testing required

This issue is classified as high risk. Mutation testing is mandatory before the pull request can pass the policy gate.

- Add or strengthen automated tests for the changed production code.
- Add `.ci/mutation.sh` if the repository does not already have one.
- The script must run a real mutation engine and exit non-zero when relevant mutants survive.
- Recommended engines: `mutmut` (Python), `gremlins` (Go), `Stryker` (JS/TS), `cargo-mutants` (Rust), PIT (JVM), Infection (PHP).
- Do not silence, exclude, or mock away changed production code merely to make the mutation gate pass.
- Equivalent or unreachable mutants may be documented explicitly; all other mutants in changed code must be killed.
- Link the pull request to this issue with `Closes #<issue>` or `Fixes #<issue>` so the gate can inherit these labels.
"""


def label_names(labels: object) -> set[str]:
    """Normalize GitHub/Forgejo label payloads into a set of names."""
    if not isinstance(labels, list):
        return set()

    names: set[str] = set()
    for label in labels:
        if isinstance(label, str):
            names.add(label)
        elif isinstance(label, dict) and isinstance(label.get("name"), str):
            names.add(label["name"])
    return names


def linked_issue_numbers(text: str | None) -> tuple[int, ...]:
    """Return same-repository issue references from a PR body, preserving order."""
    if not text:
        return ()

    seen: set[int] = set()
    numbers: list[int] = []
    for match in _LINKED_ISSUE_RE.finditer(text):
        number = int(match.group(1))
        if number not in seen:
            seen.add(number)
            numbers.append(number)
    return tuple(numbers)


def mutation_reasons(
    event: dict,
    fetch_issue: Callable[[int], dict],
) -> tuple[str, ...]:
    """Return high-risk labels inherited from the PR and its linked issues."""
    pull_request = event.get("pull_request") or {}
    labels = label_names(pull_request.get("labels"))

    for number in linked_issue_numbers(pull_request.get("body")):
        issue = fetch_issue(number)
        labels.update(label_names(issue.get("labels")))

    return tuple(label for label in MUTATION_REQUIRED_LABELS if label in labels)


def _api_request(method: str, path: str, payload: dict | None = None) -> object:
    base_url = os.environ["POLICY_API_URL"].rstrip("/")
    repository = os.environ["POLICY_REPOSITORY"]
    token = os.environ["POLICY_TOKEN"]
    provider = os.environ.get("POLICY_PROVIDER", "github").lower()

    headers = {
        "Accept": "application/json",
        "Content-Type": "application/json",
    }
    if provider == "forgejo":
        headers["Authorization"] = f"token {token}"
    else:
        headers["Authorization"] = f"Bearer {token}"
        headers["X-GitHub-Api-Version"] = "2022-11-28"

    data = json.dumps(payload).encode() if payload is not None else None
    request = Request(
        f"{base_url}/repos/{repository}/{path.lstrip('/')}",
        data=data,
        headers=headers,
        method=method,
    )
    with urlopen(request, timeout=20) as response:
        raw = response.read().decode()
    return json.loads(raw) if raw else None


def _load_event() -> dict:
    event_path = os.environ.get("POLICY_EVENT_PATH") or os.environ.get(
        "GITHUB_EVENT_PATH"
    ) or os.environ.get("FORGEJO_EVENT_PATH")
    if not event_path:
        raise RuntimeError("No workflow event path available")
    return json.loads(Path(event_path).read_text())


def _write_output(name: str, value: str) -> None:
    output_path = os.environ.get("GITHUB_OUTPUT") or os.environ.get("FORGEJO_OUTPUT")
    if output_path:
        with open(output_path, "a", encoding="utf-8") as handle:
            handle.write(f"{name}={value}\n")


def notify(event: dict) -> int:
    """Post one idempotent mutation-policy comment when a high-risk label is added."""
    label = event.get("label") or {}
    label_name = label.get("name")
    if label_name not in MUTATION_REQUIRED_LABELS:
        print(f"Label {label_name!r} does not require mutation testing")
        return 0

    issue = event.get("issue") or {}
    number = issue.get("number") or event.get("number")
    if not number:
        raise RuntimeError("Issue event does not contain an issue number")

    comments = _api_request("GET", f"issues/{number}/comments") or []
    if isinstance(comments, list) and any(
        COMMENT_MARKER in str(comment.get("body", ""))
        for comment in comments
        if isinstance(comment, dict)
    ):
        print("Mutation policy comment already present")
        return 0

    _api_request("POST", f"issues/{number}/comments", {"body": MUTATION_GUIDANCE})
    print(f"Posted mutation policy guidance on issue #{number}")
    return 0


def classify(event: dict) -> int:
    """Expose whether the PR must run mutation testing."""
    reasons = mutation_reasons(
        event,
        lambda number: _api_request("GET", f"issues/{number}") or {},
    )
    required = bool(reasons)
    _write_output("required", "true" if required else "false")
    _write_output("labels", ",".join(reasons))
    if required:
        print(f"Mutation testing required by: {', '.join(reasons)}")
    else:
        print("Mutation testing not required by issue/PR labels")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("notify", "classify"))
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    event = _load_event()
    if args.command == "notify":
        return notify(event)
    return classify(event)


if __name__ == "__main__":
    raise SystemExit(main())
