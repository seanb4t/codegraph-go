export function format(n: number) {
  return n + 1;
}
export const Label = (p: { n: number }) => {
  return <b>{format(p.n)}</b>;
};
