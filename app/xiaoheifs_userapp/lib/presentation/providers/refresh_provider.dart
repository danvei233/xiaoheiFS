import 'package:flutter_riverpod/flutter_riverpod.dart';

class RefreshEvent {
  final String route;
  final int nonce;

  const RefreshEvent({required this.route, required this.nonce});
}

final pageRefreshProvider = StateProvider<RefreshEvent?>((ref) => null);

