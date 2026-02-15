# Output Templates

Use these templates after extracting Figma data through MCP.

## Table of Contents

- Implementation Spec
- Token Table
- Component Mapping
- Design vs Code QA

## Implementation Spec

```markdown
## Scope
- File: <file-name-or-key>
- Frame/Node: <node-path> (<node-id>)
- Platform: <web/mobile/system>

## Layout
- Container: <width x height, constraints>
- Spacing: <scale and key measurements>
- Alignment rules: <left/center/right, distribution>

## Typography
- Families: <font-family list>
- Scale: <size/line-height/weight by role>
- Text behavior: <wrap, truncation, min/max lines>

## Colors and Effects
- Semantic colors: <token names and values>
- Borders/radius: <rules>
- Shadows/blur: <rules>

## Components and States
- Component: <name>
- Variants: <variant keys and options>
- States: <default/hover/focus/active/disabled>
- Interactions: <documented behavior>

## Open Questions
- <missing source information>
```

## Token Table

```markdown
| Token | Value | Type | Source Node |
|---|---|---|---|
| color.surface.default | #FFFFFF | color | Hero/Card (1:42) |
| color.text.primary | #111827 | color | Hero/Title (1:57) |
| space.300 | 12px | spacing | List/Item (2:10) |
| radius.md | 8px | radius | Button/Base (3:4) |
| text.body.md.size | 16px | typography | Body/Text (4:15) |
```

## Component Mapping

```markdown
## Component
- Name: <figma-component-name>
- Node: <node-id>

## Props
| Prop | Type | Values | Maps To |
|---|---|---|---|
| size | enum | sm, md, lg | Variant:size |
| tone | enum | primary, secondary | Variant:tone |
| disabled | boolean | true, false | State |

## Slots
- leadingIcon: optional
- label: required text
- trailingIcon: optional

## Behavior
- Default: <rules>
- Hover: <rules>
- Focus: <rules>
- Disabled: <rules>
```

## Design vs Code QA

```markdown
## Summary
- Scope: <page/frame/component>
- Result: <pass/fail/mixed>

## Findings
| Severity | Area | Figma | Code | Delta | Suggested Fix |
|---|---|---|---|---|---|
| High | Spacing | 24px top padding | 16px | -8px | Set container pt to 24px |
| Medium | Typography | 600 weight | 500 | -100 | Use semibold token |
| Low | Radius | 10px | 8px | -2px | Use radius token lg |

## Missing Inputs
- <states/assets not available in scope>

## Notes
- Mark any item as inferred when exact MCP data was unavailable.
```
