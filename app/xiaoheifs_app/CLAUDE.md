# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Xiaohei Cloud Financial Management System (小黑云财务管理系统)** - A Flutter cross-platform mobile admin console for managing a VPS cloud platform. This is the mobile/admin companion app to the web platform (located in `../../backend/` and `../../frontend/`).

**Tech Stack:**
- Flutter 3.10+ with Material Design 3
- Dart programming language
- Provider for state management
- HTTP package for API calls
- SharedPreferences for persistent storage

**Target Platforms:** Android, iOS, Windows, Linux, macOS, Web

## Development Commands

```bash
# Install dependencies
flutter pub get

# Run in development mode
flutter run

# Run on specific device
flutter run -d <device-id>

# Run Widgetbook (component playground)
flutter run -t lib/widgetbook.dart

# Build for production
flutter build apk              # Android APK
flutter build ios              # iOS
flutter build windows          # Windows executable
flutter build linux            # Linux executable
flutter build macos            # macOS
flutter build web              # Web application

# Run tests
flutter test

# Code quality
flutter analyze                # Static analysis
flutter format .               # Format code
flutter clean                  # Clean build artifacts
```

## Architecture

### State Management: Provider Pattern

The app uses a single top-level `AppState` class (`lib/app_state.dart`) that extends `ChangeNotifier`:

```dart
class AppState extends ChangeNotifier {
  Session? _session;
  bool _isReady = false;

  bool get isReady => _isReady;
  bool get isLoggedIn => _session != null;
  Session? get session => _session;
  ApiClient? get apiClient; // Lazy instantiation from session

  Future<void> load()              // Load from storage
  Future<void> loginWithApiKey()   // API Key auth
  Future<void> loginWithPassword() // Password auth
  Future<void> logout()            // Clear session
  Future<void> updateProfile()     // Update profile data
}
```

**Usage Pattern:**
```dart
// Read without rebuild
final client = context.read<AppState>().apiClient;

// Watch state changes
Consumer<AppState>(
  builder: (context, state, _) {
    if (!state.isReady) return SplashScreen();
    if (!state.isLoggedIn) return LoginScreen();
    return MainApp();
  },
)
```

### Navigation Structure

The app uses a bottom navigation bar with 5 tabs (RootScaffold):

| Tab | Screen | Purpose |
|-----|--------|---------|
| 0 | HomeScreen | Dashboard/overview |
| 1 | OrdersScreen | Order management |
| 2 | UsersScreen | User management |
| 3 | ServersScreen | VPS management |
| 4 | SettingsScreen | System settings hub |

**Navigation Patterns:**
- Main tabs: `IndexedStack` in RootScaffold preserves state
- Detail screens: `Navigator.push()` with `MaterialPageRoute`
- Return values: `Navigator.pop(context, true)` signals parent to refresh

### Directory Structure

```
lib/
├── main.dart                    # App entry point, theme configuration
├── app_state.dart               # Global state (auth, session)
├── widgetbook.dart              # Widgetbook component playground
│
├── models/
│   └── session.dart             # Session model
│
├── screens/                     # UI screens (26+ screens)
│   ├── login_screen.dart        # Login (API Key + Password)
│   ├── root_scaffold.dart       # Main navigation container
│   ├── home_screen.dart         # Dashboard
│   ├── orders_screen.dart       # Order list
│   ├── order_detail_screen.dart # Order details
│   ├── users_screen.dart        # User list
│   ├── user_detail_screen.dart  # User details
│   ├── servers_screen.dart      # VPS list
│   ├── vps_detail_screen.dart   # VPS details
│   ├── tickets_screen.dart      # Support tickets
│   ├── ticket_detail_screen.dart
│   ├── settings_screen.dart     # Settings hub
│   ├── api_keys_screen.dart
│   ├── audit_logs_screen.dart
│   ├── scheduled_tasks_screen.dart
│   ├── payment_providers_screen.dart
│   ├── permissions_screen.dart
│   ├── settings_kv_screen.dart
│   ├── wallet_orders_screen.dart
│   └── catalog/                 # Catalog management
│       ├── catalog_hub_screen.dart
│       ├── simple_crud_screen.dart  # Generic CRUD component
│       ├── regions_screen.dart
│       ├── lines_screen.dart
│       ├── packages_screen.dart
│       ├── plan_groups_screen.dart
│       ├── system_images_screen.dart
│       ├── goods_types_screen.dart
│       └── billing_cycles_screen.dart
│
└── services/                    # Business logic & API
    ├── api_client.dart          # HTTP client wrapper
    ├── app_storage.dart         # Persistent storage
    ├── admin_auth.dart          # Authentication service
    └── avatar.dart              # Avatar URL utilities
```

## Authentication

The app supports **two authentication methods**:

### 1. API Key Authentication (Simple)
- User enters: API URL + API Key + Display Name
- Direct API calls with `Authorization: Bearer <api_key>`
- No profile data loaded
- Useful for service accounts or quick access

### 2. Password Authentication (Recommended)
- User enters: API URL + Username + Password
- Calls `POST /admin/api/v1/auth/login` to get JWT token
- Fetches admin profile from `GET /admin/api/v1/profile`
- Stores username, email, permissions
- **Recommended** for full access to admin features

### Session Storage

**Primary:** SharedPreferences (cross-platform)
**Fallback:** File storage (`session.json` in app support directory)
**Web:** SharedPreferences only (no file system)

## API Integration

### ApiClient Class (lib/services/api_client.dart)

Centralized HTTP client with methods:

```dart
Future<Map<String, dynamic>> getJson(path, {query})
Future<Map<String, dynamic>> postJson(path, {body, query})
Future<Map<String, dynamic>> patchJson(path, {body, query})
Future<Map<String, dynamic>> deleteJson(path, {body, query})
```

**Features:**
- Automatic base URL normalization
- Bearer token authentication (JWT or API Key)
- JSON request/response handling
- Error handling with ApiException
- Query parameter encoding

**Error Handling Pattern:**
```dart
try {
  final resp = await client.getJson('/admin/api/v1/endpoint');
  // Process response
} catch (e) {
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(content: Text('错误：$e')),
  );
}
```

## Common Screen Patterns

### List Screen Pattern

Used in: Orders, Users, Servers, Tickets, API Keys, etc.

```dart
class XxxScreen extends StatefulWidget {
  ApiClient? _client;
  bool _loading = false;
  List<Item> _items = [];

  // Pagination
  int _page = 1;
  int _pageSize = 20;
  int _total = 0;

  // Filters
  final _keywordController = TextEditingController();
  String _statusFilter = '';

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != _client) {
      _client = client;
      _load(client);
    }
  }

  Future<void> _load(ApiClient client) async {
    setState(() => _loading = true);
    try {
      final resp = await client.getJson('/admin/api/v1/endpoint');
      // Process and attach related data
    } finally {
      setState(() => _loading = false);
    }
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) _load(client);
  }
}
```

### Detail Screen Pattern

Used in: Order Detail, User Detail, VPS Detail, Ticket Detail

```dart
class XxxDetailScreen extends StatefulWidget {
  final int itemId;

  // Return result to parent
  Future<bool> _onWillPop() async {
    Navigator.pop(context, _changed);
    return false;
  }

  @override
  Widget build(BuildContext context) {
    return WillPopScope(
      onWillPop: _onWillPop,
      child: FutureBuilder(
        future: _future,
        builder: (context, snapshot) {
          // Build UI
        },
      ),
    );
  }
}
```

### Generic CRUD Screen

File: `lib/screens/catalog/simple_crud_screen.dart`

Reusable component for basic CRUD operations:
- Configurable field definitions (text, number, boolean)
- Automatic form generation
- List view with add/edit/delete
- Used for: Regions, Lines, Packages, etc.

## UI/UX Design System

### Theme Configuration
- **Design Language:** Material Design 3 (Material You)
- **Color Scheme:** Teal/Cyan primary (`#00BFA6`)
- **Font Family:** Google Noto Sans SC (Chinese), Noto Serif SC (headings)
- **Background:** Light gray (`#F6F8FA`)
- **Card Style:** White background, 16px border radius, no elevation

### Typography Scale
- Headline Large: 32px, Noto Serif SC, W800
- Headline Medium: 26px, Noto Serif SC, W700
- Title Large: 20px, Noto Serif SC, W700
- Body: Noto Sans SC

### Common UI Patterns

**Show error:**
```dart
ScaffoldMessenger.of(context).showSnackBar(
  SnackBar(content: Text('错误：$e')),
);
```

**Navigate to detail:**
```dart
Navigator.push(
  context,
  MaterialPageRoute(builder: (_) => DetailScreen(id: item.id)),
);
```

**Refresh on return:**
```dart
final result = await Navigator.push(...);
if (result == true) _refresh();
```

## API Endpoints

The app communicates with a Go backend at `/admin/api/v1/`:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/auth/login` | POST | Admin login (JWT) |
| `/profile` | GET | Current admin profile |
| `/dashboard/overview` | POST | Dashboard statistics |
| `/orders` | GET | List orders |
| `/orders/{id}` | GET | Order details |
| `/users` | GET | List users |
| `/users/{id}` | GET | User details |
| `/vps` | GET | List VPS instances |
| `/tickets` | GET | List support tickets |
| `/wallet/orders` | GET | Wallet orders |
| `/api-keys` | GET | API key management |
| `/audit-logs` | GET | Admin activity logs |
| `/scheduled-tasks` | GET | Scheduled tasks |
| `/permissions/list` | GET | Permission registry |
| `/permissions/sync` | POST | Sync permissions |
| `/avatar/qq/{qq}` | GET | QQ avatar proxy |

## Status Mappings

### Order Statuses
`pending_review`, `rejected`, `approved`, `provisioning`, `active`, `failed`, `canceled`

### VPS Automation States
`1=running`, `2=stopped`, `3=reinstalling`, `4=resizing`, `5=renewing`, `6=error`

### VPS Admin Status
`normal`, `abuse`, `fraud`, `locked`

## Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| provider | ^6.0.0 | State management |
| http | ^1.2.2 | HTTP requests |
| shared_preferences | ^2.2.3 | Key-value storage |
| path_provider | ^2.1.4 | File system paths |
| google_fonts | ^6.2.1 | Typography |
| url_launcher | ^6.3.0 | Open external URLs |
| widgetbook | ^3.7.0 | Component development (dev) |

## Backend Integration

This Flutter app is the **mobile admin interface** for the Go backend in `../../backend/`.

**Backend Location:** `D:\项目\golang\xiaohei\backend\`

**API Documentation:**
- `backend/docs/openapi.yaml` - OpenAPI specification
- `backend/docs/frontend-readme.md` - Integration guide with status mappings

**Shared Features:**
- Same API endpoints as web admin
- Same JWT tokens
- Same permission system
- Same data models

## Code Style

**Analysis Options** (analysis_options.yaml):
- Most const/constructors lint rules are disabled for rapid development
- Consider enabling stricter rules for production

**Formatting:**
- Use `flutter format .` to format code
- Follow Dart style guide: https://dart.dev/guides/language/effective-dart/style

## Known Limitations

1. **No offline mode** - Requires active API connection
2. **No caching** - All data fetched fresh from API
3. **No background sync** - No periodic data refresh
4. **Limited error recovery** - Basic error messages only
5. **No analytics** - No usage tracking or crash reporting
6. **Test coverage** - No automated tests
7. **Hardcoded strings** - No internationalization (Chinese only)

## Adding New Features

### Adding a New Screen

1. Create screen file in `lib/screens/`
2. Extend `StatefulWidget`
3. Implement `didChangeDependencies` for client detection:
   ```dart
   @override
   void didChangeDependencies() {
     super.didChangeDependencies();
     final client = context.read<AppState>().apiClient;
     if (client != _client) {
       _client = client;
       _load(client);
     }
   }
   ```
4. Add navigation route from parent screen
5. Test with Widgetbook if reusable component

### Adding API Integration

1. Add method to `ApiClient` if needed (or use existing methods)
2. Call from screen with error handling
3. Update data model if response structure changed

### State Updates

```dart
// Local state
setState(() {
  _items = newItems;
});

// Global state
context.read<AppState>().updateProfile(...);
```

## Files to Check When Modifying

| Change | Files to Update |
|--------|----------------|
| Add new API endpoint | `lib/services/api_client.dart`, specific screen |
| Add new screen | Screen file, parent navigation |
| Change theme | `lib/main.dart` ThemeData |
| Add authentication method | `lib/app_state.dart`, `lib/screens/login_screen.dart` |
| Change storage | `lib/services/app_storage.dart` |
| Add model | Create in `lib/models/` or inline in screen |
