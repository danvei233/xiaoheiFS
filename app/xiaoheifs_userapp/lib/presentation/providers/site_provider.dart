import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/site_repository.dart';
import '../../core/utils/map_utils.dart';

class SiteState {
  final bool loading;
  final Map<String, dynamic> settings;
  final String siteName;
  final String logoUrl;
  final String faviconUrl;

  const SiteState({
    this.loading = false,
    this.settings = const {},
    this.siteName = '小黑云控制台',
    this.logoUrl = '',
    this.faviconUrl = '',
  });

  SiteState copyWith({
    bool? loading,
    Map<String, dynamic>? settings,
    String? siteName,
    String? logoUrl,
    String? faviconUrl,
  }) {
    return SiteState(
      loading: loading ?? this.loading,
      settings: settings ?? this.settings,
      siteName: siteName ?? this.siteName,
      logoUrl: logoUrl ?? this.logoUrl,
      faviconUrl: faviconUrl ?? this.faviconUrl,
    );
  }
}

final siteRepositoryProvider = Provider<SiteRepository>((ref) {
  return SiteRepository();
});

final siteProvider = StateNotifierProvider<SiteNotifier, SiteState>((ref) {
  return SiteNotifier(ref.read(siteRepositoryProvider));
});

class SiteNotifier extends StateNotifier<SiteState> {
  SiteNotifier(this._repo) : super(const SiteState());
  final SiteRepository _repo;

  Future<void> fetchSettings() async {
    state = state.copyWith(loading: true);
    try {
      final res = await _repo.getSiteSettings();
      final items = res['items'];
      final settings = <String, dynamic>{};
      if (items is List) {
        for (final item in items) {
          final map = ensureMap(item);
          final key = map['key']?.toString();
          if (key != null) {
            settings[key] = map['value'];
          }
        }
      }
      state = state.copyWith(
        loading: false,
        settings: settings,
        siteName: settings['site_name']?.toString() ?? '小黑云控制台',
        logoUrl: settings['logo_url']?.toString() ?? '',
        faviconUrl: settings['favicon_url']?.toString() ?? '',
      );
    } catch (_) {
      state = state.copyWith(loading: false);
    }
  }
}
