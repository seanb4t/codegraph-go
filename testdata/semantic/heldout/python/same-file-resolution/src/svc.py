def clean(v):
    return v


class Service:
    def run(self):
        return clean(self.step())

    def step(self):
        return 1


class Other:
    def step(self):
        return 2
