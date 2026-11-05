def wrap(f):
    return f


class C:
    @wrap
    def run(self):
        setattr(self, "computed", 1)
