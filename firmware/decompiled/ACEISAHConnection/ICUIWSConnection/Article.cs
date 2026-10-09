namespace ICUIWSConnection;

public class Article
{
	public string Name { get; set; }

	public string PartCode { get; set; }

	public string DetailCode { get; set; }

	public int Quantity { get; set; }

	public string Description { get; set; }

	public Article(string partCode, string row, int quantity, string desc)
	{
		Name = partCode;
		PartCode = partCode;
		DetailCode = row;
		Quantity = quantity;
		Description = desc;
	}
}
