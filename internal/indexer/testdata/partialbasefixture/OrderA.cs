namespace Shop;

public partial class Order : Entity
{
    public int Total()
    {
        return Id();
    }
}
