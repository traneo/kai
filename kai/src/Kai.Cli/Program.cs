using System.CommandLine;
using System.CommandLine.Parsing;
using kai.Cli.Commands;

var root = new RootCommand("kai - AI Development Lifecycle Companion");

var initCmd = new Command("init", "Scaffold kai.json configuration in the current directory");
var planCmd = new Command("plan", "Generate a plan for a development goal");
var runCmd = new Command("run", "Plan, code, and commit — full semi-autonomous pipeline");

root.Add(initCmd);
root.Add(planCmd);
root.Add(runCmd);

InitCommand.Configure(initCmd);
PlanCommand.Configure(planCmd);
RunCommand.Configure(runCmd);

return await root.Parse(args).InvokeAsync();
