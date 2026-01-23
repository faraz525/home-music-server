# CrateDrop Promo Video

Marketing video for CrateDrop, built with [Remotion](https://remotion.dev).

## Quick Start

```bash
cd video
npm install
npm run studio    # Opens Remotion Studio in browser
```

## Render Video

```bash
# 1920x1080 landscape (default)
npm run render

# 1080x1080 square (Instagram/social)
npm run render:square
```

Output will be in `out/` directory.

## Compositions

- **CrateDropPromo** - 1920x1080 @ 30fps, ~15 seconds
- **CrateDropPromo-Square** - 1080x1080 @ 30fps, ~15 seconds

## Customization

Edit `src/Root.tsx` to change:
- `brandColor` - Primary brand color (default: `#8B5CF6`)
- `accentColor` - Secondary accent (default: `#06B6D4`)

## Scenes

1. **IntroScene** - Animated logo reveal with vinyl record
2. **ProblemScene** - Pain points with strike-through animation
3. **FeatureScene** - Feature cards with staggered entry
4. **DemoScene** - Browser mockup showing the app UI
5. **OutroScene** - CTA with pulsing button
