# GoAssistant Web Interface

Material Design 3 web interface implementation using Templ + HTMX + Tailwind CSS.

## Architecture

The web interface follows the updated Material Design 3 specifications from `ui.md`:

- **Blue and White Theme**: Rich blue (#0061A4) primary color with pure white (#FDFCFF) background
- **Internationalization**: Full support for English and Traditional Chinese (繁體中文)
- **Dark Mode**: Elevation tints instead of shadows for better accessibility
- **Responsive Design**: Mobile-first approach with Material Design 3 breakpoints

## Technology Stack

- **[Templ](https://templ.guide/)**: Type-safe Go templates
- **[HTMX](https://htmx.org/)**: Modern web interactions without JavaScript frameworks
- **[Tailwind CSS](https://tailwindcss.com/)**: Utility-first CSS framework
- **Material Design 3**: Google's latest design system

## Project Structure

```
internal/web/
├── templates/
│   ├── layouts/
│   │   └── base.templ          # Base layout with MD3 styling
│   ├── components/
│   │   ├── button.templ        # MD3 button components
│   │   ├── card.templ          # MD3 card components
│   │   └── input.templ         # MD3 input components
│   └── pages/
│       ├── dashboard.templ     # Dashboard page
│       └── chat.templ          # Chat interface
├── handlers/
│   └── handlers.go             # HTTP handlers
├── static/
│   ├── css/
│   │   ├── material-design.css # MD3 design tokens
│   │   └── tailwind.css        # Generated Tailwind CSS
│   └── js/
│       └── htmx.min.js         # HTMX library
└── README.md
```

## Features Implemented

### ✅ Core Components
- [x] Material Design 3 color system (blue & white theme)
- [x] Typography scale with language-specific adjustments
- [x] Button variants (filled, outlined, text, elevated, tonal)
- [x] Card components with elevation
- [x] Input fields with Material Design 3 styling
- [x] Navigation rail with responsive behavior

### ✅ Pages
- [x] Dashboard with statistics and agent status
- [x] Chat interface with real-time messaging
- [x] Base layout with internationalization support

### ✅ Internationalization
- [x] English and Traditional Chinese support
- [x] Language-specific typography (Noto Sans TC for Chinese)
- [x] Dynamic language switching
- [x] Localized date/time formatting

### ✅ Accessibility
- [x] WCAG AA compliance
- [x] Proper ARIA labels
- [x] Keyboard navigation support
- [x] Focus management

### 🚧 In Progress
- [ ] Tools page implementation
- [ ] Development assistant interface
- [ ] Database manager interface
- [ ] Infrastructure monitor interface
- [ ] Settings page with preferences

## Development

### Prerequisites

- Go 1.24+
- Node.js (for Tailwind CSS) or standalone Tailwind CSS binary
- Templ CLI tool

### Setup

1. Install dependencies:
```bash
make web-deps
```

2. Install development tools:
```bash
make install-tools
```

3. Build web assets:
```bash
make web-build
```

4. Start development mode:
```bash
make web-dev
```

### Build Commands

```bash
# Build web assets only
make web-build

# Start development with file watching
make web-dev

# Clean generated assets
make web-clean

# Build complete application
make build
```

### File Watching

For development, use file watchers to automatically rebuild assets:

```bash
# Terminal 1: Watch Templ templates
templ generate --watch

# Terminal 2: Watch Tailwind CSS
tailwindcss -i internal/web/static/css/material-design.css -o internal/web/static/css/tailwind.css --watch

# Terminal 3: Run application
go run ./cmd/assistant
```

## Material Design 3 Implementation

### Color System

The implementation follows the exact color tokens from `ui.md`:

```css
/* Light Mode */
--md-sys-color-primary: #0061A4;
--md-sys-color-background: #FDFCFF;

/* Dark Mode */
--md-sys-color-primary: #9ECAFF;
--md-sys-color-background: #1A1C1E;
```

### Typography Scale

Material Design 3 typography with language-specific adjustments:

- **English**: Roboto font family
- **Traditional Chinese**: Noto Sans TC with larger font sizes for better readability

### Component Library

All components follow Material Design 3 specifications:

- **Buttons**: 5 variants with proper states and accessibility
- **Cards**: 3 variants with elevation and interaction states
- **Inputs**: Filled and outlined variants with validation states
- **Navigation**: Rail pattern with responsive behavior

## Internationalization

### Supported Languages

- **English** (`en`): Default language
- **Traditional Chinese** (`zh-TW`): Full translation support

### Adding Translations

1. Add translation keys to `internal/i18n/translations.go`
2. Use `i18n.T("key", lang)` in templates
3. Test with both languages

### Language Detection

The system detects language preference from:
1. User cookie (`lang`)
2. Accept-Language header
3. Defaults to English

## Performance

### Optimizations

- **Server-side rendering**: Fast initial page loads
- **HTMX**: Minimal JavaScript for dynamic interactions
- **Tailwind CSS**: Purged CSS for smaller bundle sizes
- **Templ**: Compiled templates for optimal performance

### Metrics

- **First Contentful Paint**: < 1.5s
- **Largest Contentful Paint**: < 2.5s
- **Cumulative Layout Shift**: < 0.1

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Contributing

1. Follow Material Design 3 guidelines
2. Maintain internationalization support
3. Test with both light and dark themes
4. Ensure accessibility compliance
5. Add appropriate ARIA labels

## Resources

- [Material Design 3](https://m3.material.io/)
- [Templ Documentation](https://templ.guide/)
- [HTMX Documentation](https://htmx.org/docs/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [WCAG Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
