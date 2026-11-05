package p;

class X {
  int save() {
    return write();
  }

  int write() {
    return 1;
  }

  int audit() {
    return 2;
  }

  int commit() {
    return save();
  }

  int rollback() {
    return audit();
  }
}
