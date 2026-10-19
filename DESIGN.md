---
name: go-ls
description: Filesystem Snapshot, Bottom-Up Merkle Hash & 3D Isometric Storage Telemetry Console
colors:
  telemetry-cobalt: "#1662a2"
  telemetry-cobalt-deep: "#0f4b82"
  telemetry-cobalt-soft: "#dcebfa"
  selection-royal: "#3b63eb"
  selection-royal-soft: "#eef3ff"
  signal-vermilion: "#d9532f"
  signal-crimson: "#e5484d"
  led-emerald: "#22c55e"
  led-lime: "#7ac143"
  status-added-bg: "#dcf5e6"
  status-added-fg: "#136235"
  status-modified-bg: "#fcecd9"
  status-modified-fg: "#7d4a0d"
  status-deleted-bg: "#fbe3e5"
  status-deleted-fg: "#8d1b22"
  obsidian-chassis: "#08090b"
  shopfloor-porcelain: "#eff3f6"
  drawer-ash: "#f5f6f8"
  pure-surface: "#ffffff"
  subtle-mist: "#e6ecf0"
  anodized-aluminum: "#e3e6e5"
  border-hairline: "#cfd6de"
  border-strong: "#a8b3bf"
  ink-primary: "#18202a"
  ink-secondary: "#4d5a6b"
  ink-muted: "#6f7d8f"
  rail-ivory: "#f3f5f8"
typography:
  display:
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
    fontSize: "1.4rem"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
    fontSize: "1.02rem"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "-0.02em"
  title:
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
    fontSize: "0.92rem"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "-0.01em"
  body:
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
    fontSize: "0.84rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "0.74rem"
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: "0.02em"
rounded:
  xs: "5px"
  sm: "6px"
  md: "10px"
  lg: "12px"
  xl: "14px"
  viewport: "22px"
  pill: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
components:
  button-primary:
    backgroundColor: "{colors.telemetry-cobalt}"
    textColor: "{colors.pure-surface}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "6px 14px"
  button-primary-hover:
    backgroundColor: "{colors.telemetry-cobalt-deep}"
  button-secondary:
    backgroundColor: "{colors.pure-surface}"
    textColor: "{colors.ink-primary}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "6px 14px"
  button-secondary-hover:
    backgroundColor: "{colors.subtle-mist}"
  floor-tab:
    backgroundColor: "{colors.pure-surface}"
    textColor: "{colors.ink-secondary}"
    typography: "{typography.body}"
    rounded: "{rounded.xs}"
    padding: "6px 14px"
  floor-tab-active:
    backgroundColor: "{colors.shopfloor-porcelain}"
    textColor: "{colors.ink-primary}"
  operation-card:
    backgroundColor: "{colors.pure-surface}"
    textColor: "{colors.ink-primary}"
    rounded: "{rounded.lg}"
    padding: "12px 14px"
  operation-card-selected:
    backgroundColor: "{colors.selection-royal-soft}"
    textColor: "{colors.ink-primary}"
    rounded: "{rounded.lg}"
    padding: "12px 14px"
  hash-chip:
    backgroundColor: "{colors.subtle-mist}"
    textColor: "{colors.ink-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.xs}"
    padding: "2px 8px"
  drive-chip:
    backgroundColor: "{colors.telemetry-cobalt-soft}"
    textColor: "{colors.telemetry-cobalt}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "2px 9px"
---

# Design System: go-ls

## Overview

**Creative North Star: "The Isometric Hardware Telemetry Deck"**

`go-ls` is an industrial-technical storage telemetry and Merkle verification console that bridges low-level filesystem inspection with tactile spatial visualization. The interface frames a crisp, high-density light porcelain workspace inside an obsidian-dark instrument chassis (`#08090b`), pairing scannable operational route cards on the left with full-bleed 3D isometric hardware modules and hierarchical WinDirStat trees on the right. Every visual element is grounded in hardware and cryptographic reality: directories appear as extruded anodized-aluminum modules on an isometric shopfloor grid with corner rivets, cooling vents (`//////`), 5-dot capacity LEDs, and radar target rings, while operational cards summarize host-to-drive storage routes, revision deltas (`v1`, `v2`), and circular volume share rings.

The aesthetic philosophy is crisp, industrial-technical, and high-density. It explicitly rejects decorative marketing gradients, neon cyberpunk dark-mode glow, and ungrounded floating cards in favor of physical hardware metaphors, flush structural split panes, and monospace precision.

**Key Characteristics:**
- **Obsidian Instrument Framing:** Fixed left navigation rail (`64px` collapsed, `248px` expanded) and `8px` top/bottom dark frame borders that enclose a rounded light viewport (`22px` left corners, flush right edge).
- **Card-Style Top Header & Operational Split Workspace:** A rounded top header card in normal document flow (`.viewport-header`, `margin: 1rem 1.4rem 0 1.4rem`) with a global search pill and contextual segmented options sitting above the left card drawer (`Reported Modules` / `Tracked Subtrees`) and right stage.
- **3D Isometric Shopfloor Telemetry:** Extruded 3D cushion modules rendered in painter's order inside `#treemap-workspace` (fitting `100%` of the remaining vertical screen height) over a `30°` isometric coordinate grid with hatched floor shadow pads, floating HUD controls, and interactive zoom.
- **Cryptographic Monospace Readouts:** All file paths, CRC-32/SHA-256 digests, drive UUIDs, byte capacities, and revision tags are set in `JetBrains Mono` for immediate scannability.

## Colors

The palette contrasts an obsidian outer instrument frame and crisp porcelain workspace surfaces against high-visibility industrial telemetry accents (cobalt, vermilion, and LED emerald).

### Primary
- **Telemetry Cobalt** (`{colors.telemetry-cobalt}`): Used for primary action buttons, active tree/subtree percentage bars, drive identity chips, and primary interactive focus rings.
- **Telemetry Cobalt Deep** (`{colors.telemetry-cobalt-deep}`): Hover and active state for primary action triggers.
- **Selection Royal** (`{colors.selection-royal}`): Used for selected operation card borders (`2px solid`), active dock buttons, and HUD node badges.

### Secondary
- **Signal Vermilion** (`{colors.signal-vermilion}`): Used on the 3D Isometric Cushion Map for active radar target rings, 5-dot wall capacity LEDs, and revision version tags (`v2`, `DIR`).
- **Signal Crimson** (`{colors.signal-crimson}`): Used for active floor/mode indicator dots (`.op-floor-dot`), top-rank operation card badges, and circular progress ring strokes.

### Tertiary
- **LED Emerald** (`{colors.led-emerald}`): Used for the API Online status orb (`16px` circle with a `2px solid #ffffff` border) and verified status text (`Subtree Ready`, `Active`).
- **LED Lime** (`{colors.led-lime}`): Used for top-surface hardware corner status LEDs on directory modules in the 3D Isometric Cushion Map.
- **Semantic Diff Tokens** (`{colors.status-added-bg}` / `{colors.status-added-fg}`, `{colors.status-modified-bg}` / `{colors.status-modified-fg}`, `{colors.status-deleted-bg}` / `{colors.status-deleted-fg}`): Used exclusively for snapshot revision deltas (`+Added`, `~Modified`, `-Deleted`).

### Neutral
- **Obsidian Chassis** (`{colors.obsidian-chassis}`): Outer app shell frame (`8px` top and bottom borders) and fixed left navigation rail background.
- **Shopfloor Porcelain** (`{colors.shopfloor-porcelain}`): Main viewport canvas background and active segmented tab fill.
- **Drawer Ash** (`{colors.drawer-ash}`): Background of the left collapsible operations drawer (`.op-sidebar-panel`).
- **Pure Surface** (`{colors.pure-surface}`): Primary surface for operation cards, data tables, sticky top header bar, and floating HUD pills.
- **Anodized Aluminum** (`{colors.anodized-aluminum}`): Extruded 3D isometric module walls (`#e3e6e5` to `#d9dddc` left wall, `#ecefed` to `#e1e5e3` right wall) and technical shopfloor grid tones.
- **Border Hairline & Strong** (`{colors.border-hairline}`, `{colors.border-strong}`): `1px solid` structural dividers between the top header, left drawer, tables, and cards.
- **Ink Hierarchy** (`{colors.ink-primary}`, `{colors.ink-secondary}`, `{colors.ink-muted}`): Primary headings/data cells, secondary metadata, and muted labels.

### Named Rules
**The Hardware-Grounded Color Rule.** High-chroma colors are reserved strictly for telemetry signals: active tab dots (`{colors.signal-crimson}`), 3D module LEDs (`{colors.signal-vermilion}`, `{colors.led-lime}`), API health orbs (`{colors.led-emerald}`), and file-extension category swatches. Structural chrome remains neutral porcelain, ash, and obsidian.

**The White-Ringed Status Orb Rule.** The API health indicator in the bottom sidebar rail must always render as a `16px` circle (green `#22c55e` when online, red `#ef4444` when offline) enclosed by a crisp `2px solid #ffffff` border and subtle status glow.

## Typography

**Display & UI Font:** `Inter` (with `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif` fallback)
**Label & Telemetry Mono Font:** `JetBrains Mono` (with `ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace` fallback)

**Character:** Clean geometric sans-serif UI chrome paired with high-legibility technical monospace for every filesystem path, byte metric, drive UUID, and Merkle digest.

### Hierarchy
- **Display** (`700`, `1.4rem`, `line-height: 1.2`, `letter-spacing: -0.02em`): Dashboard KPI metric values (`Tracked Computers`, `Indexed Volume`).
- **Headline** (`700`, `1.02rem`, `line-height: 1.25`, `letter-spacing: -0.02em`): Sticky top viewport header title (`3D Cushion Map`, `Directory Tree`).
- **Title** (`700`, `0.92rem`, `line-height: 1.25`): Left drawer header title (`Reported Modules`, `Tracked Subtrees`) and surface panel titles.
- **Body** (`400`–`600`, `0.84rem`, `line-height: 1.5`): Standard UI controls, button labels, and table descriptions.
- **Label / Mono Telemetry** (`600`–`700`, `0.68rem`–`0.82rem`, `line-height: 1.35`, `JetBrains Mono`): Operation card titles, Merkle hash chips, drive UUID badges, byte readouts, and 3D isometric wall labels (`matrix(0.8660254, 0.5, 0, 1, 0, 0)`).

### Named Rules
**The Monospace Telemetry Rule.** Any string representing a filesystem path, file extension, byte count, percentage share, revision number (`v1`), drive UUID, or Merkle hash (`crc32`/`sha256`) must be set in `JetBrains Mono`.

## Layout

The application uses a **Framed Sidebar + Operational Split Workspace** spatial model:

1. **App Shell (`.app-shell`):** Full-viewport flex row (`100vh`, `overflow: hidden`) with `background: #08090b` and `padding: 8px 0` (preserving `8px` dark borders at the top and bottom while keeping the right edge borderless).
2. **Fixed Left Sidebar Rail (`.sidebar-rail`):** Sticky vertical rail (`width: 64px` collapsed, `248px` expanded, `height: calc(100vh - 16px)`) with smooth cubic-bezier width transition (`0.24s cubic-bezier(0.22, 1, 0.36, 1)`). Scrolling inside the main viewport never moves the sidebar.
3. **Main Viewport (`.main-viewport`):** Rounded left corners (`border-radius: 22px 0 0 22px`), `height: calc(100vh - 16px)`, independent vertical scroll with custom floating pill scrollbars (`10px` width, `999px` thumb radius).
4. **Card-Style Top Header (`.viewport-header`):** Renders as a rounded card in normal document flow (`margin: 1rem 1.4rem 0 1.4rem`, `padding: 0.7rem 1.15rem`, `border-radius: 14px`, `border: 1px solid var(--border-default)`), housing the view title on the left, global search pill (`max-width: 520px`) in the center, and contextual segmented option bar (`.op-top-option-bar`) + drawer toggle action on the right.
5. **Operational Split Workspace (`.op-workspace-split` & `#treemap-workspace`):** Sits beneath `.viewport-header` (`grid-template-columns: clamp(300px, 24vw, 356px) minmax(0, 1fr)`, `gap: 1rem`, `margin: 1rem 1.4rem 1.4rem 1.4rem`). On the 3D Cushion Map tab, `#treemap-workspace` fits the remaining vertical viewport height (`height: calc(100vh - 116px)`, `min-height: 500px`, `margin-bottom: 0.6rem`) with `.iso-cushion-stage` at `height: 100%` so the 3D canvas fits the screen vertically without scrolling. Collapses smoothly to `1fr` when `.drawer-collapsed` is active or below `1024px` viewport width.
6. **Dashboard-Only Overview Metrics (`.metrics-strip`):** The 5-column KPI overview strip (`grid-template-columns: repeat(auto-fit, minmax(190px, 1fr))`) appears exclusively on the **Dashboard** tab (`activeTab === 'dashboard'`) inside `.dashboard-viewport-body`.

## Elevation & Depth

`go-ls` uses a **Structural & Tactile Hybrid** depth system. Workspace panes and the `.viewport-header` card are separated by crisp `1px solid` structural borders (`#cfd6de`), while floating stage HUD controls use elevated ambient shadows, and 3D Cushion Map modules project physical isometric extrusion (`32px`–`55px` Z-height) over hatched floor shadow pads (`url(#iso-hatch-pattern)`).

### Shadow Vocabulary
- **Subtle Surface Rest** (`box-shadow: 0 1px 3px rgba(15, 23, 42, 0.06)`): Default resting shadow for dashboard metric boxes and cards.
- **Operation Card Selected** (`box-shadow: 0 6px 16px rgba(79, 125, 243, 0.14)`): Elevated blue-tinted focus shadow when an operation card is active in the left drawer.
- **Floating Stage HUD** (`box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12)`): Applied to the bottom-center host floor bar (`.op-floating-bottom-bar`), bottom-left breadcrumb HUD, bottom-right zoom dock, and floating extension legend card.
- **Isometric Module Extrusion** (`filter: drop-shadow(0 8px 6px rgba(30, 36, 43, 0.10))` + hatched SVG floor pad): Grounds 3D hardware blocks onto the isometric shopfloor grid.

### Named Rules
**The Card Header & Vertical-Fit Stage Rule.** `.viewport-header` renders as a rounded card in normal document flow (`margin: 1rem 1.4rem 0 1.4rem`), `#treemap-workspace` fits `100%` of the remaining vertical screen height (`height: calc(100vh - 116px)`), and stage overlays (host floor selector, breadcrumb pill, zoom dock, legend card, and drawer toggle) float cleanly over the stage with drop shadows.

## Shapes

The form language pairs a macro-rounded outer viewport shell (`22px` top-left and bottom-left radius) with crisp, instrument-grade inner geometry:
- **Viewport Shell:** `22px 0 0 22px` left-side rounding against the dark sidebar rail; `0px` on the right edge.
- **Operation Cards & Floating Legend:** `12px` border radius with `1.5px solid #e5e8ef` resting borders (`2px solid #4f7df3` when selected).
- **Segmented Option Bars (`.op-top-option-bar`, `.op-floating-bottom-bar`):** `9px`–`10px` outer container radius with `3px`–`4px` inner padding, containing `7px`-radius tab pills.
- **Buttons, Inputs & Chips:** `6px`–`8px` radius for standard buttons and search pills; `5px` for monospace hash chips; `999px` full pill for drive UUID badges and custom scrollbar thumbs.
- **3D Isometric Modules:** True `30°` isometric rhombuses (`cos(30°) = 0.8660254`, `sin(30°) = 0.5`) with `1.25px`–`1.65px` dark charcoal strokes (`#4a4f55` resting, `#1f2328` selected) and circular dark action button (`54px` diameter, `border-radius: 50%`).

## Components

Components feel **instrument-grade, tactile, and telemetry-dense**.

### Buttons
- **Shape:** Gently rounded technical corners (`6px` radius for `.btn`, `7px` for `.op-floor-tab` and `.op-icon-btn`).
- **Primary (`.btn-primary`):** Telemetry Cobalt fill (`{colors.telemetry-cobalt}`), pure white text (`#ffffff`), `0.38rem 0.8rem` padding, `500` weight. Darkens to `{colors.telemetry-cobalt-deep}` on hover.
- **Secondary (`.btn`):** Pure white background (`#ffffff`), `1px solid {colors.border-hairline}`, dark ink text (`{colors.ink-primary}`), shifting to `{colors.subtle-mist}` on hover.
- **Floating Circular Action Button (`.iso-floating-arrow-btn`):** `54px × 54px` circle (`border-radius: 50%`), `#1e2226` fill, white `→` arrow icon, scaling `1.06x` on hover.

### Segmented Option Bars (`.op-top-option-bar` & `.op-floating-bottom-bar`)
- **Style:** Horizontal pill group (`#ffffff` in the top header bar, `rgba(255, 255, 255, 0.96)` with `backdrop-filter: blur(8px)` at the bottom center of the stage), `1px solid {colors.border-hairline}`, `9px`–`10px` radius, `3px`–`4px` padding.
- **State:** Active tab (`.op-floor-tab.active`) displays a `6px` Signal Crimson dot (`#e5484d`), `#f1f4f9` fill, and `700` font weight.

### Operation Cards (`.op-card`)
- **Corner Style:** `12px` radius (`1.5px solid #e5e8ef` default, `2px solid #4f7df3` with `#f3f6ff` background when `.selected`).
- **Internal Structure:** Three-tier telemetry layout:
  1. **Top Row (`.op-card-top`):** `21px` square telemetry icon badge (`.op-card-icon-badge` with `⚡` / `▣` glyphs; `#17191e` dark, `#e5484d` red, or `#3b63eb` blue), monospace title, vermilion version pill (`v2` / `DIR`), and `22px` SVG circular progress ring.
  2. **Route Connector Row (`.op-card-route`):** Left origin host/status (`Subtree Ready` in green `#16a34a`), center horizontal route line (`1.5px` `#334155` with `4px` solid dot and `8px` hollow ring node), and right target drive UUID + vermilion byte delta (`+412.5 MB`).
  3. **Footer Row (`.op-card-footer`):** Monospace metadata pills (`◫ 24 files`, `▣ 6 dirs`) and right-aligned percentage share badge (`32%`).

### Inputs / Fields
- **Global Search Pill (`.viewport-search-pill`):** Centered in `.viewport-header` (`max-width: 520px`), `#f3f5f8` background, `8px` radius, `0.38rem 0.8rem` padding. Transitions to `#ffffff` background, `{colors.telemetry-cobalt}` border, and `0 0 0 3px rgba(24, 104, 183, 0.12)` focus ring on `:focus-within`.
- **Compact Scope Selects (`.op-compact-select`):** Custom SVG chevron select (`7px` radius, `0.78rem` font size, `600` weight) inside the left operations drawer header.

### Navigation (`.sidebar-rail`)
- **Style:** Fixed dark vertical rail (`#08090b`), `44px`-tall `.rail-item` buttons with `12px` radius, centered `20px` stroke icons when collapsed (`64px`), and smooth label/count reveal when expanded (`248px`).
- **Collapse Toggle (`#sidebar-toggle-btn`):** Uses directional double-chevron arrows (`»` when collapsed, `«` when expanded).
- **Bottom Utility Stack (`.sidebar-bottom`):** Houses the **Sync Server** button (`#sync-server-btn`), the **Collapse Sidebar** toggle (`#sidebar-toggle-btn`), the total **Indexed Volume** storage badge, and the white-bordered green/red `.status-orb` API health indicator (`#server-health-badge`).

### 3D Isometric Cushion Map (`.iso-cushion-stage`)
- **Signature Component:** Interactive SVG stage (`viewBox="0 0 1020 620"`, `height: 100%` inside vertical-fit `#treemap-workspace`) rendering recursive treemap subdivisions as extruded 3D hardware blocks on a `30°` isometric grid. Each block includes a hatched floor shadow pad, left-front wall with isometric monospace title (`#1 /SRC`) and angled cooling vents (`//////`), right-front wall with 5 capacity LED dots, top cushion surface with 3 dark corner rivets + 1 color-coded status LED, and concentric dashed vermilion radar rings when selected.

## Do's and Don'ts

### Do:
- **Do** place all tab-specific view options (`.op-top-option-bar`) inside the rounded top header card (`.viewport-header` in normal document flow, `margin: 1rem 1.4rem 0 1.4rem`) and keep `Sync Server` in `.sidebar-bottom`.
- **Do** keep the 5-card overview KPI strip (`.metrics-strip`) exclusively on the **Dashboard** tab (`activeTab === 'dashboard'`).
- **Do** maintain the `8px` top and bottom dark frame borders on `.app-shell` with zero right-side border (`padding: 8px 0 8px 0`).
- **Do** use directional double-chevron arrows (`»` to expand, `«` to collapse) on `#sidebar-toggle-btn` and a `2px solid #ffffff` border on the green/red `.status-orb`.
- **Do** render all hashes, file paths, drive UUIDs, and byte counts in `JetBrains Mono` (`var(--font-mono)`).

### Don't:
- **Don't** allow scrolling in `.main-viewport` to scroll or shift the fixed left `.sidebar-rail`.
- **Don't** make `.viewport-header` `position: sticky` or `position: fixed`, and don't let `#treemap-workspace` overflow vertically—keep `#treemap-workspace` fitted to the remaining screen height (`height: calc(100vh - 116px)`) with floating HUD overlays for the legend, breadcrumb pill, host floor bar, and `+`/`−` zoom controls.
- **Don't** introduce purple-to-blue AI gradients, dark-mode neon glow on the workspace canvas, or ungrounded decorative charts disconnected from real scan metadata.
