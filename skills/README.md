# Example Skills

This directory contains example skills that demonstrate the skill format and can be used for testing.

## Using these examples

To add an example skill to your agm repository:

```bash
agm
# Select "repo" → "Add skill" → "Local Directory"
# Enter the path: /path/to/agent-management/skills/example-skill
```

## Creating your own skills

Every skill must have a `SKILL.md` file at its root. This file contains:

1. **Title** - What the skill does
2. **Instructions** - How the AI should use this skill
3. **Examples** (optional) - Sample usage

Example structure:

```
my-skill/
└── SKILL.md
```

Once created, add it to agm and link it to any project that needs it.
