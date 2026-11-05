// é
function plain() {
  return 1;
}
const expr = function () {
  return plain();
};
const alias = plain;
const [one, , two] = [1, 2, 3];
class Counter {
  total = 0;
  bump() {
    return expr();
  }
}
