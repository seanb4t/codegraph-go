pub struct Circle {
    r: i32,
}

pub enum Kind {
    Round,
}

pub trait Shape {
    fn area(&self) -> i32;
    fn describe(&self) -> i32 {
        0
    }
}

impl Shape for Circle {
    fn area(&self) -> i32 {
        3 * self.r * self.r
    }
}

impl Circle {
    pub fn unit() -> Circle {
        Circle { r: 1 }
    }
}

mod inner {
    pub struct Hidden;

    pub fn hidden() -> i32 {
        1
    }
}

pub fn top_level() -> i32 {
    fn local() -> i32 {
        2
    }
    local()
}
