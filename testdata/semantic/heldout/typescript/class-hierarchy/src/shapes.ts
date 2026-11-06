export class Base {
  name() {
    return "base";
  }
}
export class Square extends Base {
  describe() {
    return this.name();
  }
}
export class Other {
  name() {
    return "other";
  }
}
export function use() {
  const s = new Square();
  return s.describe();
}
