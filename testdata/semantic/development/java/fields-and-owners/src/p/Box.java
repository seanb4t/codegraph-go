package p;

// é 😀
public class Box implements Shape {
  int width, height = 2;
  static final String NAME = "é";
  int size;

  public int area() {
    return mul(width, height);
  }

  int mul(int a, int b) {
    return a * b;
  }

  int size() {
    return size;
  }
}
