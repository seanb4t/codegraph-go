"""é"""
LIMIT, OTHER = 1, 2
value: int


def helper():
    return LIMIT


class Base:
    kind = "base"

    def __init__(self):
        self.name = helper()

    def rename(self, name):
        self.name = name


class Child(Base):
    def __init__(self):
        self.value = helper()
