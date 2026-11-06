import { helper as h, other } from "./lib";

export function caller() {
  return h() + other();
}

export function shadow() {
  const other = () => 3;
  return other();
}
