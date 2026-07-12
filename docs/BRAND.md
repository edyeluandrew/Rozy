# Rozy brand & design tokens

## CSS variables (admin / web)

```css
:root {
  --kd-cream: #FDF9ED;
  --kd-beige: #F0E0B1;
  --kd-sand: #EBE5D7;
  --kd-gold: #EAC333;
  --kd-dark-gold: #6E5505;
  --kd-black: #020201;
  --kd-charcoal: #2A2925;
  --kd-grey: #85867E;
  --kd-border: #C2C1BF;
  --kd-white: #FCFCFB;
}
```

## Semantic usage

| Role | Token | Hex |
|------|-------|-----|
| **App background** | `--kd-cream` | `#FDF9ED` |
| **Primary buttons / CTAs** | `--kd-gold` | `#EAC333` |
| **Premium sections / dark cards** | `--kd-black` | `#020201` |
| **Body text** | `--kd-charcoal` | `#2A2925` |
| **Driver app header / royal mode** | `--kd-dark-gold` | `#6E5505` |
| **Light cards / surfaces** | `--kd-white` | `#FCFCFB` |
| **Secondary surfaces** | `--kd-sand` | `#EBE5D7` |
| **Accent / highlights** | `--kd-beige` | `#F0E0B1` |
| **Muted text** | `--kd-grey` | `#85867E` |
| **Borders / dividers** | `--kd-border` | `#C2C1BF` |

## App-specific notes

### Passenger app (Rozy)
- Background: cream
- Primary CTA: gold on charcoal text (ensure contrast)
- Cards: white on cream

### Driver app (Rozy Driver)
- Header / app bar: dark gold (`#6E5505`)
- Background: cream
- Wallet / earnings highlights: black premium cards with gold accents

### Admin dashboard
- Same tokens via `admin/src/index.css`
- Dark sidebar optional: `--kd-black` with gold active states

## Flutter

See `mobile/lib/core/theme/rozy_colors.dart` and `rozy_theme.dart`.

## Accessibility

- Gold `#EAC333` on cream: use **charcoal or black text** on gold buttons, not white.
- Dark gold header: use **cream or white** text.
