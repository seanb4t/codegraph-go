def save():
    return write()


def write():
    return 1


def audit():
    return 2


def commit():
    return save()


def rollback():
    return audit()
