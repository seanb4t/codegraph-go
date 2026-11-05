package p;

public class A {
  public static int keep() {
    return gone();
  }

  static int gone() {
    return 1;
  }
}
