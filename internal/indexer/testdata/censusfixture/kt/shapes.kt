package shapes

interface Shape {
    fun area(): Int
}

class Circle(val r: Int) : Shape {
    override fun area(): Int = 3 * r * r

    class Inner {
        fun inside(): Int = 1
    }

    companion object {
        fun unit(): Circle = Circle(1)
    }
}

enum class Kind {
    ROUND;

    fun label(): String = name
}

object Registry {
    fun count(): Int = 0
}

fun topLevel(): Int {
    fun local(): Int = 2
    return local()
}
