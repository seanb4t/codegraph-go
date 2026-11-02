protocol Shape {
    func area() -> Int
}

struct Circle: Shape {
    let r: Int

    func area() -> Int { return 3 * r * r }

    struct Inner {
        func inside() -> Int { return 1 }
    }

    static func + (lhs: Circle, rhs: Circle) -> Circle { return lhs }
}

extension Circle {
    func perimeter() -> Int { return 6 * r }
}

func topLevel() -> Int {
    func local() -> Int { return 2 }
    return local()
}
