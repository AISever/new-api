---
name: new-api-frontend-style
description: Use when creating or updating React pages, Semi UI views, dashboard panels, settings screens, storefront flows, or empty/error states in this new-api repository and the goal is to match the established visual language, spacing, theming, and interaction style instead of inventing a new UI direction.
---

# new-api Frontend Style

## Overview

Keep this repository on its existing React 18 + Semi UI visual system. Reuse the current page language: card-led layouts, restrained blue/cyan emphasis, Semi design tokens, dark-mode support, and practical interaction patterns that favor readable tables, forms, tags, banners, and empty states over decorative novelty.

## Use This Skill

- Read [references/visual-language.md](references/visual-language.md) before editing `web/src/components/` or `web/src/pages/`.
- Match the page type first:
  - Console/admin/settings pages stay utilitarian: stacked `Card` sections, compact spacing, minimal gradients.
  - User-facing commerce pages can be richer: rounded hero cards, paired light/dark gradients, stronger CTA hierarchy.
- Keep existing route shells and padding conventions from `PageLayout.jsx`; do not invent a competing layout container.

## Core Rules

- Prefer Semi primitives already common in the repo: `Card`, `Tabs`, `Table`, `Descriptions`, `Banner`, `Empty`, `Space`, `Tag`, `Typography`.
- Use `Text` and `Paragraph` for supporting copy. Use `Title` sparingly, usually once per page or major card.
- Drive color through `var(--semi-color-*)` tokens. If custom gradients are needed, provide both light and dark variants.
- Keep radii and depth consistent with the repo: admin cards are modest; consumer cards often use `rounded-2xl` or `rounded-[28px]` with soft shadows.
- Preserve bilingual/i18n discipline: user-facing copy goes through `t('...')`, and helper text should be short, directive, and concrete.
- Use inline empty/error states for page-owned failures. If the component owns the failure UI, bypass the global API error interceptor and render the state locally.
- Sensitive data stays masked by default and uses explicit reveal actions.

## Page Checklist

- Is this page an admin/settings surface or a user-facing transactional surface?
- Does spacing follow existing `p-4 md:p-6`, `md:px-6`, `space-y-*`, and max-width patterns nearby?
- Are status colors semantic and reused through `Tag` or shared maps rather than ad hoc text colors?
- Does dark mode have an explicit counterpart for any custom background or border choice?
- Are error, loading, and empty states styled consistently with the surrounding page?

## Common Mistakes

- Do not drop a bright new palette, new font stack, or glassmorphism pattern into one page.
- Do not hardcode colors when a Semi token or existing status map already expresses the state.
- Do not mix toast-driven and inline error handling in the same flow.
- Do not add oversized hero sections to settings/admin pages.
- Do not hide critical actions inside ornamental layouts; the repo favors obvious actions over novelty.
