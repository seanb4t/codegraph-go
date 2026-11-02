<?php
namespace Shapes;

interface Shape {
    public function area(): int;
}

class Circle implements Shape {
    public function area(): int { return 3; }
}

trait Named {
    public function name(): string { return "x"; }
}

enum Kind {
    case Round;

    public function label(): string { return "round"; }
}

if (!function_exists('Shapes\helper')) {
    function helper(): int { return 1; }
}

function top_level(): int {
    $f = new class { public function inside(): int { return 1; } };
    function nested(): int { return 2; }
    return nested();
}
