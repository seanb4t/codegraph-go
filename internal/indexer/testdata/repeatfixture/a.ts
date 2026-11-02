export const e = () => {};

export function measure(n: number): number {
  return n * 2;
}

describe("geometry", () => {
  it("first", () => {
    function nextGeometry() {
      return 1;
    }
    nextGeometry();
  });

  it("second", () => {
    function nextGeometry() {
      return measure(2);
    }
    nextGeometry();
  });

  it("third", () => {
    function nextGeometry() {
      return 3;
    }
    nextGeometry();
  });

  it("nested e", () => {
    function e() {
      return 0;
    }
    function callNested() {
      return e();
    }
    callNested();
  });

  it("first model", () => {
    class Model {
      load() {
        return 1;
      }
    }
    new Model().load();
  });

  it("second model", () => {
    class Model {
      save() {
        return 2;
      }
    }
    new Model().save();
  });

  it("helpers", () => {
    [1].forEach(() => { function helper() { return 1; } helper(); }); [2].forEach(() => { function helper() { return 2; } helper(); });
  });
});
