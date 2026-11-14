package p;

public class Order {
  int quantity;
  static final String CURRENCY = "EUR";

  enum State {
    OPEN,
    CLOSED
  }
}
