class InputLimits {
  static const int username = 64;
  static const int email = 254;
  static const int phone = 32;
  static const int qq = 32;
  static const int bio = 512;
  static const int intro = 1024;
  static const int password = 128;

  static const int ticketSubject = 240;
  static const int ticketContent = 10000;
  static const int resourceName = 200;

  static const int paymentMethod = 64;
  static const int paymentNote = 1000;
  static const int approval = 1000;
  static const int screenshotUrl = 1024;
}

int runeLength(String text) => text.runes.length;
