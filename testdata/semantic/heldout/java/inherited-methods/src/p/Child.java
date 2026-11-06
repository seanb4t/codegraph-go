package p;

class Child extends Base {
  int use() {
    return name() + Tools.twice(2);
  }
}

class Tools {
  static int twice(int v) {
    return v * 2;
  }
}
