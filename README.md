# Agent skill management

A single-binary CLI that manages AI agent skills across multiple tools. Import once, symlink everywhere.

If you use Cursor, Claude, Codex, Copilot, or Gemini — and you're tired of manually copying prompt files between projects — this tool fixes that.

## What it does

```
GitHub / Registry / Local folder
        |
        v
~/.agent-management/repo/         <-- imported once
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

### 1. Sync from a team registry (recommended)

If your team maintains a shared skills repo:

```bash
agm
# Select "Sync skills"
# First time: paste the registry URL (e.g. https://github.com/ArdentaCorp/skills)
# It clones the repo and imports all skills automatically
```

To pull updates later, just run "Sync skills" again — or use the non-interactive flag:

```bash
agm --sync
```

Sync will:

- Pull the latest changes from the registry
- Add any new skills
- Update changed skills
- Remove skills that were deleted from the registry
- Replace duplicate same-name skills from other sources (`github:*`, `local:*`) with the registry version to keep one canonical entry

### 2. Import individual skills

```bash
agm
# Select "Import skills"
```

You can import from:

- **GitHub Repository** — paste any GitHub URL:

  ```
  https://github.com/user/repo              # full clone
  https://github.com/user/repo/tree/main/x  # sparse checkout of /x only
  ```

- **Local Folder** — point to a directory on disk and pick which skills to import:
  ```
  # agm scans the folder for subdirectories containing SKILL.md
  # and lets you multi-select which ones to import
  ```

Every skill must contain a `SKILL.md` file or it will be rejected.

### 3. Link skills to a project

```bash
cd ~/my-project    # must have .cursor/, .claude/, .codex/, etc.
agm
# Select "Link to project"
# Pick a tool (or "All detected tools") > toggle skills with Space > Enter
```

This creates symlinks from your global repo into the project's skill directory. Different projects can have different skill sets.

### 4. Manage skills

```bash
agm
# Select "Manage skills" > pick a skill > Update or Delete
```

Update fetches the latest commits and pulls changes. All symlinked projects get the update automatically.

## Supported AI tools

| Tool           | Detected via                                      |
| -------------- | ------------------------------------------------- |
| Cursor         | `.cursor/`                                        |
| Claude         | `.claude/`                                        |
| Codex          | `.codex/` or `.agents/`                           |
| GitHub Copilot | `.copilot/` or `.github/`                         |
| Antigravity    | `.gemini/antigravity/global_skills/` or `.agent/` |

Skills get symlinked into a `skills/` subdirectory of whichever is detected (e.g. `.cursor/skills/my-skill`).

## How it works internally

### Data directory

Everything lives in `~/.agent-management/`:

```
~/.agent-management/
├── config.json                        # registry URL, system info, custom tools
├── registry/                          # cloned registry repo (via sync)
└── repo/
    ├── skills.json                    # registry of all installed skills
    ├── registry__my-skill/            # synced from registry
    ├── github__user__repo/            # full clone
    ├── github__user__repo__subdir/    # sparse checkout
    └── local__my-skill/               # imported from local folder
```

Skill IDs use the format `registry:name`, `github:user/repo/path`, or `local:name`. They get encoded to safe directory names by replacing `/` and `:` with `__`.

When linked to a project, symlinks use just the skill name (e.g. `my-skill`, not `registry__my-skill`).

### Registry

A registry is just a GitHub repo with skills as subdirectories:

```
your-skills-repo/
├── code-review/
│   └── SKILL.md
├── testing/
│   └── SKILL.md
└── deployment/
    └── SKILL.md
```

Set it once with `agm` > "Sync skills", and the URL is saved in `config.json`. Every `agm --sync` after that pulls changes and keeps your skills current.

### Symlinks

- **macOS / Linux**: standard symlinks
- **Windows**: directory junctions (no admin privileges needed)

### Custom tool support

Add your own AI tools by editing `~/.agent-management/config.json`:

```json
{
  "system": "windows",
  "registry": "https://github.com/ArdentaCorp/skills",
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
agm --sync       # sync from registry (non-interactive)
agm --help       # usage info
agm --version    # print version
agm --config     # show current configuration
```

## License

MIT
