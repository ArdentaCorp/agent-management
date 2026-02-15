# Agent skill management

A single-binary CLI that manages AI agent skills across multiple tools. Clone once, symlink everywhere.

If you use Cursor, Claude, Codex, or Copilot — and you're tired of manually copying prompt files between projects — this tool fixes that.

## What it does

```
GitHub / Local directory
        |
        v
~/.agent-management/repo/         <-- cloned once
        |
        +-- symlink --> project-a/.cursor/skills/
        +-- symlink --> project-a/.claude/skills/
        +-- symlink --> project-b/.codex/skills/
```

One source of truth. All your projects stay in sync. Edit a skill once, every linked project sees the change instantly.

## Install

### With Go (recommended)

```bash
go install github.com/ArdentaCorp/agent-management/cmd/agm@latest
```

This puts `agm` in your `$GOPATH/bin`. Make sure that's in your `PATH`.

### From source

```bash
git clone https://github.com/ArdentaCorp/agent-management.git
cd agent-management
make install   # or: go install ./cmd/agm
```

### Download binary

Grab a binary from [Releases](https://github.com/ArdentaCorp/agent-management/releases) and put it somewhere in your PATH.

### Build for other platforms

```bash
GOOS=darwin  GOARCH=arm64 go build -o agm-darwin-arm64 ./cmd/agm
GOOS=linux   GOARCH=amd64 go build -o agm-linux-amd64 ./cmd/agm
GOOS=windows GOARCH=amd64 go build -o agm.exe ./cmd/agm
```

### Update

```bash
go install github.com/ArdentaCorp/agent-management/cmd/agm@latest
```

## Quick start

### 1. Add a skill from GitHub

```bash
agm
# Select "repo" > "Add skill" > "GitHub Repository"
# Paste: https://github.com/anthropics/skills/tree/main/skills/pdf
```

Only the `skills/pdf` subdirectory gets downloaded (sparse checkout). No need to clone the entire repo.

You can also add full repos or local directories:

```
https://github.com/user/repo              # full clone
https://github.com/user/repo/tree/main/x  # sparse checkout of /x
~/Desktop/my-custom-skill                  # local copy
```

Every skill must contain a `SKILL.md` file or it will be rejected.

### 2. Link skills to a project

```bash
cd ~/my-project    # must have .cursor/, .claude/, .codex/, etc.
agm
# Select "list(cursor)" > toggle skills with Space > Enter
```

This creates symlinks from your global repo into the project's skill directory. Different projects can have different skill sets.

### 3. Update skills

```bash
agm
# Select "repo" > pick a skill > "Update"
```

Fetches the latest commits and pulls changes. All symlinked projects get the update automatically.

## Supported AI tools

| Tool           | Detected directories                              |
| -------------- | ------------------------------------------------- |
| Antigravity    | `.gemini/antigravity/global_skills/` or `.agent/` |
| GitHub Copilot | `.copilot/` or `.github/`                         |
| Cursor         | `.cursor/`                                        |
| Claude         | `.claude/`                                        |
| Codex          | `.codex/` or `.agents/`                           |

Skills get installed into the `skills/` subdirectory of whichever is detected (e.g. `.cursor/skills/`).

## How it works internally

### Data directory

Everything lives in `~/.agent-management/`:

```
~/.agent-management/
├── config.json                        # system info + optional custom tool config
└── repo/
    ├── skills.json                    # registry of all installed skills
    ├── github__user__repo/            # full clone
    └── github__user__repo__subdir/    # sparse checkout
```

Skill IDs like `github:user/repo/path` get encoded to safe directory names by replacing `/` and `:` with `__`.

### Symlinks

- **macOS / Linux**: standard symlinks
- **Windows**: directory junctions (no admin privileges needed)

### Custom tool support

Add your own AI tools by editing `~/.agent-management/config.json`:

```json
{
  "system": "windows",
  "aiTools": [
    { "type": "cursor", "skillDirs": [".cursor/skills"] },
    { "type": "my-tool", "skillDirs": [".my-tool/prompts"] }
  ]
}
```

This replaces the default tool list entirely. Include any defaults you want to keep.

## CLI flags

```
agm              # interactive mode
agm --help       # usage info
agm --version    # print version
agm --config     # show current configuration
```

## License

MIT
