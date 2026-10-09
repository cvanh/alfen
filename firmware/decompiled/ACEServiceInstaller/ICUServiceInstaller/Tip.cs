namespace ICUServiceInstaller;

public class Tip
{
	public string Id { get; }

	public string Name { get; }

	public string LabelText { get; }

	public string Tooltip { get; }

	public Tip(string id, string name, string labelText, string tooltip)
	{
		Id = id;
		Name = name;
		LabelText = labelText;
		Tooltip = tooltip;
	}
}
