#!/usr/bin/env python3
import argparse
import json
import re
import subprocess
import sys
import urllib.request
from typing import Iterable, List, Optional, Tuple


VERSION_RE = re.compile(
    r"^v(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)"
    r"(?:-patch\.(?P<patch_no>\d+))?"
    r"(?:-(?P<suffix>.+))?$"
)


def is_prerelease(tag: str) -> bool:
    lowered = tag.lower()
    return any(marker in lowered for marker in ("alpha", "beta", "rc", "preview", "pre"))


def parse_version(tag: str) -> Optional[Tuple[int, int, int, int, str]]:
    match = VERSION_RE.match(tag)
    if not match:
        return None
    major = int(match.group("major"))
    minor = int(match.group("minor"))
    patch = int(match.group("patch"))
    patch_no = int(match.group("patch_no") or 0)
    suffix = match.group("suffix") or ""
    return (major, minor, patch, patch_no, suffix)


def sort_tags(tags: Iterable[str]) -> List[str]:
    parsed = []
    for tag in tags:
        version = parse_version(tag)
        if version is None:
            continue
        parsed.append((version, tag))
    parsed.sort(key=lambda item: item[0])
    return [tag for _, tag in parsed]


def fetch_github_releases(repo: str) -> List[dict]:
    url = f"https://api.github.com/repos/{repo}/releases"
    request = urllib.request.Request(url, headers={"User-Agent": "Codex-new-api-release-workflow"})
    with urllib.request.urlopen(request, timeout=20) as response:
        return json.load(response)


def fetch_git_tags(remote: str) -> List[str]:
    result = subprocess.run(
        ["git", "ls-remote", "--tags", remote],
        check=True,
        capture_output=True,
        text=True,
    )
    tags = set()
    for line in result.stdout.splitlines():
        parts = line.split()
        if len(parts) != 2:
            continue
        ref = parts[1]
        if not ref.startswith("refs/tags/"):
            continue
        tag = ref.replace("refs/tags/", "").replace("^{}", "")
        tags.add(tag)
    return sort_tags(tags)


def first_matching(items: Iterable[str], predicate) -> Optional[str]:
    for item in items:
        if predicate(item):
            return item
    return None


def main() -> int:
    parser = argparse.ArgumentParser(description="Summarize upstream new-api releases for production decisions.")
    parser.add_argument("--repo", default="QuantumNous/new-api", help="GitHub repository in owner/name form")
    parser.add_argument("--remote", default="upstream", help="Git remote used for tag fallback")
    parser.add_argument("--recent", type=int, default=5, help="How many recent stable releases to print")
    args = parser.parse_args()

    github_releases: List[dict] = []
    github_error = None
    try:
        github_releases = fetch_github_releases(args.repo)
    except Exception as exc:  # noqa: BLE001
        github_error = str(exc)

    git_tags: List[str] = []
    git_error = None
    try:
        git_tags = fetch_git_tags(args.remote)
    except Exception as exc:  # noqa: BLE001
        git_error = str(exc)

    latest_release = github_releases[0]["tag_name"] if github_releases else None
    latest_stable_release = first_matching(
        (release["tag_name"] for release in github_releases),
        lambda tag: not is_prerelease(tag),
    )
    recent_stable_releases = [
        release["tag_name"]
        for release in github_releases
        if not is_prerelease(release["tag_name"])
    ][: args.recent]

    latest_tag = git_tags[-1] if git_tags else None
    latest_stable_tag = first_matching(reversed(git_tags), lambda tag: not is_prerelease(tag))

    recommended = latest_stable_release or latest_stable_tag

    print(f"github.latest_release={latest_release or 'unavailable'}")
    print(f"github.latest_stable_release={latest_stable_release or 'unavailable'}")
    print(f"git.latest_tag={latest_tag or 'unavailable'}")
    print(f"git.latest_stable_tag={latest_stable_tag or 'unavailable'}")
    print(f"recommended_for_production={recommended or 'unavailable'}")
    if recent_stable_releases:
        print("recent_stable_releases=" + ", ".join(recent_stable_releases))
    if github_error:
        print(f"github_error={github_error}")
    if git_error:
        print(f"git_error={git_error}")
    if recommended:
        print("reason=Prefer the newest non-alpha stable release/tag for production.")
        return 0

    print("reason=No stable upstream version could be determined.")
    return 1


if __name__ == "__main__":
    sys.exit(main())
