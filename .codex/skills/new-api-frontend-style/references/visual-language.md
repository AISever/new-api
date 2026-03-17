# Visual Language Reference

## Page Families

### 1. Console / Admin / Settings

Use the settings and admin pages as the baseline for operational surfaces:

- Representative files:
  - `web/src/components/settings/PaymentSetting.jsx`
  - `web/src/pages/Setting/Payment/SettingsExternalShopLDXP.jsx`
  - `web/src/pages/Setting/Payment/SettingsGPTTeamPlan.jsx`
- Visual pattern:
  - vertical stack of `Card` containers
  - low visual noise
  - form-first layout with `Space`, `Form`, `Input`, `Select`, `Table`
  - feedback via `Tag`, `Banner`, concise helper `Text`
- Preferred styling:
  - simple margins/padding
  - Semi tokens over custom hex
  - clear headings, little decorative chrome

### 2. User-Facing Transactional Flows

Use the shop and payment pages as the reference for richer surfaces:

- Representative files:
  - `web/src/components/shop/index.jsx`
  - `web/src/components/shop/OrderDetail.jsx`
  - `web/src/components/shop/OrderPay.jsx`
  - `web/src/components/shop/GptTeamPlanTab.jsx`
- Visual pattern:
  - large rounded cards
  - selective gradients for hero/product areas
  - stronger CTA buttons
  - descriptive empty states and masked sensitive values
- Preferred styling:
  - `rounded-2xl` to `rounded-[28px]`
  - soft shadows, never harsh borders everywhere
  - dark-mode companion backgrounds when light mode uses gradients

## Color And Theme Rules

- Base all neutral text, borders, and fills on `var(--semi-color-*)`.
- Blue/cyan is the dominant accent family in the current repo. Reuse it unless the surrounding page already establishes another semantic color.
- Status colors stay semantic:
  - pending/info: blue
  - waiting/warn: orange
  - success/delivered: green
  - failure/error: red
  - inactive/expired/default: grey
- If a page needs custom gradients, define both light and dark values in the component and branch from `useActualTheme()`.

## Layout Rules

- Respect `PageLayout.jsx` padding and shell decisions before adding wrappers.
- Transactional pages usually live in `max-w-5xl` or `max-w-7xl` containers.
- Common spacing patterns:
  - page wrapper: `p-4 md:p-6` or `px-4 pb-6 pt-0 md:px-6 md:pt-10`
  - vertical rhythm: `space-y-4` or `space-y-6`
  - grid cards: `gap-4` or `gap-5`

## Interaction Rules

- Use toasts for global success and simple validation hints.
- Use inline state blocks for page-owned failures, missing resources, or partial results.
- When rendering inline failure states after API calls, set `skipErrorHandler: true` and handle the error in the component.
- Mask email/contact/code-like secrets by default and let users reveal them intentionally.
- Prefer explicit button labels like `保存设置`, `检查结果`, `继续支付`, `查看详情`.

## Copy Rules

- Keep copy concise and operational.
- User-facing guidance should explain the next action, not narrate the UI.
- Empty states should distinguish:
  - not configured / not open
  - load failure
  - no data yet

## Quick Do / Don't

- Do extend existing status maps and helper utilities before inventing new ones.
- Do reuse nearby page structure when adding a sibling screen or tab.
- Do keep tables readable and actions visible.
- Do not mix radically different card radii, shadows, and button hierarchies on one page.
- Do not introduce decorative sections unless the page is clearly a storefront or onboarding surface.
