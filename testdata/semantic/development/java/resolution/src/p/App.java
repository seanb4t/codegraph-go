package p;

import q.Util;

public class App {
  int run() {
    return Util.helper() + local();
  }

  int local() {
    return 2;
  }
}
