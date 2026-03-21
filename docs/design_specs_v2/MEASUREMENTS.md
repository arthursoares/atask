# Exact Measurements — Extracted from HTML Reference

All values extracted from `atask-screens-validation.html`. This is the ground truth.

## CSS Variables (Design Tokens)

### Colors
| Token | Value |
|-------|-------|
| canvas | #f6f5f2 |
| canvas-elevated | #fefefe |
| canvas-sunken | #eceae7 |
| sidebar-bg | rgba(238,236,231,0.72) |
| sidebar-hover | rgba(0,0,0,0.04) |
| sidebar-active | rgba(0,0,0,0.06) |
| sidebar-selected | rgba(70,112,160,0.10) |
| ink-primary | #222120 |
| ink-secondary | #686664 |
| ink-tertiary | #a09e9a |
| ink-quaternary | #c8c6c2 |
| ink-on-accent | #fff |
| accent | #4670a0 |
| accent-hover | #3a5f8a |
| accent-subtle | rgba(70,112,160,0.10) |
| today-star | #c88c30 |
| today-bg | rgba(200,140,48,0.08) |
| someday-tint | #8878a0 |
| deadline-red | #c04848 |
| success | #4a8860 |
| agent-tint | #7868a8 |
| border | rgba(0,0,0,0.06) |
| border-strong | rgba(0,0,0,0.12) |
| separator | rgba(0,0,0,0.05) |

### Spacing (4px base)
| Token | Value |
|-------|-------|
| sp-1 | 4px |
| sp-2 | 8px |
| sp-3 | 12px |
| sp-4 | 16px |
| sp-5 | 20px |
| sp-6 | 24px |
| sp-8 | 32px |

### Radii
| Token | Value |
|-------|-------|
| radius-xs | 4px |
| radius-sm | 6px |
| radius-md | 8px |
| radius-lg | 12px |
| radius-xl | 16px |
| radius-full | 9999px |

### Font Sizes
| Token | Value |
|-------|-------|
| text-xs | 11px |
| text-sm | 12px |
| text-base | 14px |
| text-md | 15px |
| text-lg | 17px |
| text-xl | 20px |

### Layout
| Token | Value |
|-------|-------|
| sidebar-width | 240px |
| toolbar-height | 52px |

---

## Component Measurements

### App Frame
- `display: flex; width: 100vw; height: 100vh`

### Sidebar
- Width: `240px` (sidebar-width)
- Background: `sidebar-bg` with `backdrop-filter: blur(28px) saturate(160%)`
- Border-right: `1px solid border`

### Sidebar Item
- Gap: `12px` (sp-3)
- Padding: `5px 12px` (5px sp-3)
- Border-radius: `6px` (radius-sm)
- Font-size: `14px` (text-base)
- Color: `ink-secondary`
- Active: bg `sidebar-active`, color `ink-primary`, font-weight `700`

### Sidebar Group Label (Area headers)
- Font-size: `11px` (text-xs)
- Font-weight: `700`
- Color: `ink-tertiary`
- Text-transform: `uppercase`
- Letter-spacing: `0.8px`
- Padding: `8px 12px 4px` (sp-2 sp-3 sp-1)

### Sidebar Badge
- Font-size: `11px` (text-xs)
- Color: `ink-tertiary`
- Margin-left: `auto`
- Min-width: `18px`

### Sidebar Dot (project color)
- Width/height: `8px`
- Border-radius: `50%`

### Sidebar Separator
- Height: `1px`
- Background: `separator`
- Margin: `8px 16px` (sp-2 sp-4)

---

### Toolbar
- Height: `52px` (toolbar-height)
- Padding: `0 24px` (0 sp-6)
- Border-bottom: `1px solid separator`
- Background: `canvas`
- Justify-content: `space-between`

### Toolbar View Title
- Font-size: `20px` (text-xl)
- Font-weight: `700`
- Color: `ink-primary`
- Gap: `8px` (sp-2) between icon and text

### Toolbar Subtitle
- Font-size: `12px` (text-sm)
- Color: `ink-tertiary`

### Toolbar Button
- Width/height: `30px`
- Border-radius: `6px` (radius-sm)
- Color: `ink-tertiary`
- Hover: bg `sidebar-hover`, color `ink-primary`

---

### App Content (scrollable area)
- Padding: `24px` (sp-6) — ALL SIDES

---

### Task Row (collapsed)
- Height: `32px`
- Gap: `12px` (sp-3)
- Padding: `6px 16px` (6px sp-4)
- Border-radius: `8px` (radius-md)
- Hover: bg `sidebar-hover`
- Selected: bg `sidebar-selected`

### Task Title
- Font-size: `14px` (text-base)
- Color: `ink-primary`
- Completed: color `ink-tertiary`, text-decoration `line-through`, decoration-color `ink-quaternary`

### Task Title Input (editing)
- Font-size: `14px` (text-base)
- Color: `ink-primary`
- Placeholder: `ink-quaternary`

### Task Meta
- Font-size: `11px` (text-xs)
- Color: `ink-tertiary`
- Gap: `8px` (sp-2)
- Margin-left: `auto`

### Task Project Pill
- Gap: `3px`
- Background: `canvas-sunken`
- Padding: `1px 7px`
- Border-radius: `9999px` (radius-full)

### Task Meta Dot (project color in meta)
- Width/height: `6px` (implicit from pill)

---

### Checkbox
- Width/height: `20px`
- Border: `1.5px solid ink-quaternary`
- Border-radius: `50%`
- Background: `canvas-elevated`
- Checked: border `accent`, bg `accent`
- Today: border `today-star`
- Checkmark SVG: `11px`, stroke `ink-on-accent`, stroke-width `2.5`

---

### Inline Editor (Editing state)
- Height: `auto` (min-height 32px)
- Padding: `6px 16px 8px` (6px sp-4 8px)
- Background: `sidebar-selected` (accent 10%)
- Border: `1.5px solid accent`
- Border-radius: `8px` (radius-md)
- Editing top row: height `32px`, gap `12px` (sp-3)

### Attribute Bar (bottom of inline editor)
- Gap: `6px`
- Padding: `4px 0 0 27px` (left = checkbox 20 + gap ~7)

### Attribute Pill
- Font-size: `11px` (text-xs)
- Font-weight: `700`
- Padding: `2px 8px`
- Border-radius: `9999px` (radius-full)
- Border: `1px solid border`
- Variants:
  - Today: bg `today-bg`, color `today-star`
  - Project: bg `canvas-sunken`, color `ink-secondary`
  - Tag: bg `accent-subtle`, color `accent`
  - Add: bg none, color `ink-tertiary`, border dashed

---

### New Task Row
- Height: `32px` — same as task row
- Gap: `12px` (sp-3)
- Padding: `6px 16px` (6px sp-4) — same as task row
- Color: `ink-tertiary`
- Hover: color `accent`, bg `accent-subtle`
- Plus circle: `20px`, border `1.5px dashed`

---

### Section Header
- Gap: `8px` (sp-2)
- Padding: `12px 0 4px` (sp-3 0 sp-1)
- Title: font-size `14px` (text-base), weight `700`, color `ink-primary`
- Count: font-size `11px` (text-xs), color `ink-tertiary`
- Line: height `1px`, bg `separator`, flex `1`

---

### When Popover
- Width: `260px`
- Position: `top: 100%; left: 27px; margin-top: 6px`
- Background: `canvas-elevated`
- Border: `1px solid border-strong`
- Border-radius: `12px` (radius-lg)
- Shadow: `shadow-popover`

---

### Detail Panel (right side)
- Width: `340px`
- Background: `canvas-elevated`
- Border-left: `1px solid border`

### Detail Field
- Margin-bottom: `16px` (sp-4)
- Label: font-size `11px` (text-xs), weight `700`, color `ink-tertiary`, uppercase, letter-spacing `0.5px`, margin-bottom `4px` (sp-1)
- Value: font-size `12px` (text-sm), color `ink-secondary`

---

### Command Palette
- Width: `560px`
- Position: `top: 18%`
- Border-radius: `16px` (radius-xl)
- Shadow: `shadow-popover`
- Border: `1px solid border-strong`
- Input wrap: padding `12px 16px` (sp-3 sp-4), border-bottom `separator`
- Input: font-size `17px` (text-lg)
- Item: gap `12px`, padding `6px 16px`, font-size `14px` (text-base)
- Item icon: width `20px`
- Group label: `11px` bold uppercase

---

### Inbox Hover Actions
- Position: absolute right, centered vertically
- Button: `26px × 26px`, radius `6px` (sm), border `1px solid border`, bg `canvas-elevated`
- Today hover: color `today-star`, border `today-star`, bg `today-bg`

---

### Activity Entry
- Gap: `12px` (sp-3)
- Padding: `8px 0` (sp-2 0)
- Avatar: `24px × 24px`, radius `50%`, font-size `10px` bold
- Author: `11px` (text-xs) bold, color `ink-primary`
- Text: `11px` (text-xs), color `ink-secondary`, line-height relaxed
- Time: `10px`, color `ink-tertiary`
- Agent card: margin-top `8px`, bg `agent-bg`, border `1px agent-border`, radius `8px`, padding `8px 12px`

### Checklist Item
- Gap: `8px` (sp-2)
- Padding: `3px 0`
- Font-size: `12px` (text-sm)
- Color: `ink-primary`
- Checkbox: `16px × 16px`, radius `4px` (radius-xs), border `1.5px solid ink-quaternary`
- Done text: color `ink-tertiary`, line-through

### Detail Panel
- Width: `340px`
- Background: `canvas-elevated`
- Border-left: `1px solid border`
- Header: padding `20px 20px 12px` (sp-5 sp-5 sp-3), border-bottom `separator`
- Title: `17px` (text-lg) bold, color `ink-primary`
- Meta row: gap `8px`, margin-top `8px`
- Body: padding `16px 20px` (sp-4 sp-5), overflow-y auto
- Field label: `11px` (text-xs) bold uppercase, letter-spacing `0.5px`, color `ink-tertiary`, margin-bottom `4px`
- Field value: `12px` (text-sm), color `ink-secondary`
- Field margin-bottom: `16px` (sp-4)

### Tag Pill (generic)
- Font-size: `11px` (text-xs) bold
- Padding: `2px 8px`
- Border-radius: `9999px` (full)
- Gap: `4px`
- Variants:
  - default: bg `canvas-sunken`, color `ink-secondary`
  - today: bg `today-bg`, color `today-star`
  - deadline: bg `deadline-bg`, color `deadline-red`
  - agent: bg `agent-bg`, color `agent-tint`
  - success: bg `success-bg`, color `success`
  - someday: bg `rgba(155,138,191,0.08)`, color `someday-tint`
  - accent: bg `accent-subtle`, color `accent`
  - cancelled: bg `canvas-sunken`, color `ink-tertiary`

### Empty State
- Padding: `80px 32px` (sp-20 sp-8)
- Text: `15px` (text-md), color `ink-tertiary`
- Icon: `48px`, color `ink-quaternary`, opacity `0.5`, margin-bottom `16px`

### Progress Bar (Project view toolbar)
- Bar: width `80px`, height `4px`, bg `canvas-sunken`, radius `full`
- Fill: bg `accent`, radius `full`
- Text: `11px` (text-xs), color `ink-tertiary`, bold

### Date Group (Upcoming/Logbook)
- Margin-bottom: `16px` (sp-4)
- Header: `12px` (text-sm) bold, color `ink-primary`, padding `8px 16px 4px`

### Sidebar Toolbar (traffic lights area)
- Height: `52px` (toolbar-height)
- Padding: `0 16px` (0 sp-4)

### When Popover Internals
- Option: gap `12px`, padding `5px 16px`, font-size `14px` (text-base)
- Calendar: padding `8px 12px`
- Calendar header: `11px` bold, color `ink-tertiary`
- Calendar day: `12px` (text-sm), padding `3px 0`, radius `4px` (xs), color `ink-secondary`
- Icon: `20px`, font-size `14px`
- Clear: `12px` (text-sm) bold, color `ink-secondary`, padding `8px`

### Command Palette
- Width: `560px`
- Top: `18%`
- Border-radius: `16px` (radius-xl)
- Border: `1px solid border-strong`
- Shadow: `shadow-popover`
- Input wrap: padding `12px 16px` (sp-3 sp-4), gap `12px`, border-bottom `separator`
- Input: `17px` (text-lg)
- Results: padding `8px 0` (sp-2 0)
- Group label: `11px` bold uppercase, letter-spacing `0.8px`, padding `8px 16px 4px`
- Item: gap `12px`, padding `6px 16px`, font-size `14px` (text-base), color `ink-secondary`
- Item icon: `20px`
- Item shortcut: `11px` (text-xs), color `ink-tertiary`, monospace
