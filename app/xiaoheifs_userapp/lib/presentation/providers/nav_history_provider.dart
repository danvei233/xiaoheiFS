import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Navigation history state
/// Tracks the navigation stack to determine back behavior
class NavHistoryState {
  final List<String> history;

  const NavHistoryState({this.history = const []});

  NavHistoryState copyWith({List<String>? history}) {
    return NavHistoryState(history: history ?? this.history);
  }
}

/// Navigation history notifier
class NavHistoryNotifier extends StateNotifier<NavHistoryState> {
  NavHistoryNotifier() : super(const NavHistoryState());

  /// Push a new page to history.
  /// Duplicate consecutive routes are ignored.
  void push(String route) {
    if (state.history.isNotEmpty && state.history.last == route) {
      return;
    }
    final newHistory = List<String>.from(state.history)..add(route);
    state = state.copyWith(history: newHistory);
  }

  /// Reset history to a single route.
  void resetTo(String route) {
    state = NavHistoryState(history: [route]);
  }

  /// Pop current route and return the previous route.
  /// If there is only current route, keep it and return null.
  String? pop() {
    if (state.history.length <= 1) return null;
    final newHistory = List<String>.from(state.history)..removeLast();
    state = state.copyWith(history: newHistory);
    return newHistory.last;
  }

  /// Clear history
  void clear() {
    state = const NavHistoryState();
  }

  /// Check if history is empty (directly opened from bottom nav)
  bool get isEmpty => state.history.isEmpty;

  /// Check if history has a previous route.
  bool get hasPreviousRoute => state.history.length > 1;

  /// Get the previous route in history
  String? get previousRoute {
    if (state.history.length >= 2) {
      return state.history[state.history.length - 2];
    }
    return null;
  }
}

final navHistoryProvider =
    StateNotifierProvider<NavHistoryNotifier, NavHistoryState>((ref) {
      return NavHistoryNotifier();
    });
