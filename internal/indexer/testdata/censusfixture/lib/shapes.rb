module Shapes
  class Circle
    def area
      3
    end

    def self.unit
      new
    end
  end

  class Circle
    def perimeter
      6
    end
  end

  def self.helper
    1
  end
end

def top_level
  Shapes::Circle.unit
end

[1].each do |x|
  def inside_block
    x
  end
end
