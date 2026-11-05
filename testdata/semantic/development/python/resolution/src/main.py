from lib import helper as h, other


def caller():
    return h() + other()


def shadow(other):
    return other()


def dynamic(target):
    return getattr(target, "run")()


CB = caller
