// héllo 😀
export const { left, right: renamed } = { left: 1, right: 2 };
export const arrow = (n: number) => n + 1;
export let counter = 0;
export class Box {
  field = 1;
  static label = "é";
  readonly doubled = arrow(2);
  get() {
    return arrow(this.field);
  }
}
export interface Shape {
  area(): number;
}
export type Alias = number;
export function plain() {
  return new Box().get();
}
