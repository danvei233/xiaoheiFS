class InputLimits {
  static const int email = 120;
  static const int phone = 32;
  static const int qq = 20;
  static const int bio = 500;

  static const int ticketSubject = 240;
  static const int ticketContent = 10000;
  static const int resourceName = 200;

  static const int paymentMethod = 50;
  static const int paymentNote = 500;
  static const int approval = 500;
  static const int screenshotUrl = 500;
}

int runeLength(String text) => text.runes.length;
