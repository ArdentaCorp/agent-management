---
name: figma-mcp
description: Inspect and translate Figma designs through an MCP server, including reading files and nodes, extracting styles and variables, defining design tokens, and producing implementation-ready specs or design-vs-code QA reports. Use when requests include Figma links, file keys, node IDs, requests to convert Figma to code, requests for spacing or typography or color specs, or requests to compare implemented UI against Figma.
---

# Figma MCP

## Overview

Use this skill to turn Figma artifacts into clear engineering outputs with MCP tools.
Prefer deterministic extraction first, then inference, and label inferred details explicitly.

## Workflow

1. Confirm scope and output format.
2. Inspect the design through MCP calls.
3. Normalize findings into tokens, layout rules, and component behavior.
4. Produce one of the standard outputs from `references/output-templates.md`.
5. Flag assumptions, unknowns, and follow-up questions.

## Step 1: Confirm Scope

Collect the minimum required input:
- Figma URL or file key.
- Node IDs, frame names, or page names to scope analysis.
- Expected output:
  - implementation spec
  - token extraction
  - component mapping
  - design-vs-code QA report
- Target platform:
  - web React
  - mobile
  - design system documentation

If scope is ambiguous, ask for the smallest clarifying detail that unlocks progress (usually frame + output type).

## Step 2: Inspect Through MCP

Use the MCP server to fetch:
- file metadata and page list
- frame and node trees for requested scope
- text styles, color styles, effect styles, grids, and variables
- component and variant definitions
- exported asset metadata when requested

Prefer exact values from MCP output over visual guesswork.
When a value is unavailable, mark it as inferred.

## Step 3: Normalize Design Data

Convert raw design data into implementation language:
- Layout:
  - container dimensions
  - spacing scale
  - padding/margins
  - alignment
  - constraints or auto-layout behavior
- Typography:
  - font family
  - size scale
  - line height
  - weight
  - letter spacing
- Color and effects:
  - semantic roles (primary, surface, border, danger)
  - opacity states
  - elevation and blur rules
- Components:
  - props and variants
  - interactive states (default, hover, focus, active, disabled)
  - content rules and truncation behavior

Map each value to platform-ready tokens when possible.

## Step 4: Produce Output

Select a template from `references/output-templates.md`:
- `Implementation Spec`
- `Token Table`
- `Component Mapping`
- `Design vs Code QA`

Keep each statement traceable to a frame or node path.
Include node IDs in output whenever possible for auditability.

## Step 5: Quality Bar

Before finalizing:
- verify measurements and typography scale consistency
- check token naming consistency and semantic grouping
- separate extracted facts from inferred implementation choices
- list missing inputs required for exact implementation

## Guardrails

Do not claim pixel-perfect parity if assets, states, or constraints are missing.
Do not invent hidden interactions without stating that they are assumptions.
Do not overfit to one framework unless the user requested it.

## Quick Triggers

Use this skill immediately when prompts look like:
- "Turn this Figma frame into a React component spec."
- "Extract tokens from this Figma file."
- "Map this Figma component set to props and variants."
- "Compare our implemented UI to this Figma and report differences."
- "Give exact spacing and typography from this Figma node."

## References

Use `references/output-templates.md` for standardized response structures.
