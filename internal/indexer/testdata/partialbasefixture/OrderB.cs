namespace Shop;

public partial class Order : Entity
{
    public int Count()
    {
        return Total();
    }
}
