using System.CommandLine;
using System.CommandLine.Parsing;
using kai.Cli;
using kai.Cli.Commands;

Logo.Print();

var root = new RootCommand("kai-code - AI Development Lifecycle Companion");

var initCmd = new Command("init", "Scaffold kai.json configuration in the current directory");
var runCmd = new Command("run", "Plan, code, and commit — full semi-autonomous pipeline");

root.Add(initCmd);
root.Add(runCmd);

InitCommand.Configure(initCmd);
RunCommand.Configure(runCmd);

return await root.Parse(args).InvokeAsync();
