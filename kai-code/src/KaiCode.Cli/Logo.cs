namespace kai.Cli;

public static class Logo
{
    private static readonly string[] Art =
    [
        "  █  █  ██  ████  ███  ██  ███  ████",
        "  █ █  █  █   █  █    █  █ █  █ █",
        "  ██   ████   █  █    █  █ █  █ ███",
        "  █ █  █  █   █  █    █  █ █  █ █",
        "  █  █ █  █ ████  ███  ██  ███  ████",
    ];

    public static void Print()
    {
        foreach (var line in Art)
        {
            Console.WriteLine(line);
        }
        Console.WriteLine();
    }
}
