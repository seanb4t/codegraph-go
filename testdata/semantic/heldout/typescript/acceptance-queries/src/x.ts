export function save() {
  return write();
}
function write() {
  return 1;
}
export function audit() {
  return 2;
}
export function commit() {
  return save();
}
export function rollback() {
  return audit();
}
