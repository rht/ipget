/*
Package cli provides a framework to build command line applications in Go with most of the burden of arguments parsing and validation
placed on the framework instead of the user.


Basics

You start by creating an application by passing a name and a description:

	cp = cli.App("cp", "Copy files around")

To attach the code to execute when the app is launched, assign a function to the Action field:
	cp.Action = func() {
		fmt.Printf("Hello world\n")
	}

Finally, in your main func, call Run on the app:

	cp.Run(os.Args)

Options

To add a (global) option, call one of the (String[s]|Int[s]|Bool)Opt methods on the app:

	recursive := cp.BoolOpt("R recursive", false, "recursively copy the src to dst")

* The first argument is a space seperated list of names for the option without the dashes

* The second parameter is the default value for the option

* The third parameter is the option description, as will be shown in the help messages

There is also a second set of methods Bool, String, Int, Strings and Ints, which accepts structs describing the option:

	recursive = cp.Bool(BoolOpt{
		Name:  "R",
		Value: false,
		Desc:  "copy src files recursively",
		EnvVar: "",
	})

The field names are self-describing.
There EnvVar field is a space separated list of environment variables names to be used to initialize the option.

The result is a pointer to a value that will be populated after parsing the command line arguments.
You can access the values in the Action func.

In the command line, mow.cli accepts the following syntaxes

* For boolean options:

	-f : a single dash for the one letter names
	-f=false : a single dash for the one letter names, equal sign followed by true or false
	--force :  double dash for longer option names
	-it : mow.cli supports option folding, this is equivalent to: -i -t

* For string, int options:

	-e=value : single dash for one letter names, equal sign followed by the value
	-e value : single dash for one letter names, space followed by the value
	-Ivalue : single dash for one letter names immediately followed by the value
	--extra=value : double dash for longer option names, equal sign followed by the value
	--extra value : double dash for longer option names, space followed by the value

* For slice options (StringsOpt, IntsOpt): repeat the option to accumulate the values in the resulting slice:

	-e PATH:/bin -e PATH:/usr/bin : resulting slice contains ["/bin", "/usr/bin"]

Arguments

To accept arguments, you need to explicitly declare them by calling one of the (String[s]|Int[s]|Bool)Arg methods on the app:

	src := cp.StringArg("SRC", "", "the file to copy")
	dst := cp.StringArg("DST", "", "the destination")

* The first argument is the argument name as will be shown in the help messages

* The second parameter is the default value for the argument

* The third parameter is the argument description, as will be shown in the help messages

There is also a second set of methods Bool, String, Int, Strings and Ints, which accepts structs describing the argument:

	src = cp.Strings(StringsArg{
		Name:  "SRC",
		Desc:  "The source files to copy",
		Value: "",
		EnvVar: "",
	})

The field names are self-describing.
The Value field is where you can set the inital value for the argument.

EnvVar accepts a space separated list of environment variables names to be used to initialize the argument.


The result is a pointer to a value that will be populated after parsing the command line arguments.
You can access the values in the Action func.



Operators

The -- operator marks the end of options.
Everything that follow will be treated as an argument,
even if starts with a dash.

For example, given the touch command which takes a filename as an argument (and possibly other options):


	file := cp.StringArg("FILE", "", "the file to create")


If we try to create a file named -f this way:


	touch -f

Would fail, because -f will be parsed as an option not as an argument.
The fix is to prefix the filename with the -- operator:


	touch -- -f



Commands

mow.cli supports nesting commands and sub commands.
Declare a top level command by calling the Command func on the app struct, and a sub command by calling
the Command func on the command struct:

	docker := cli.App("docker", "A self-sufficient runtime for linux containers")

	docker.Command("run", "Run a command in a new container", func(cmd *cli.Cmd) {
		// initialize the run command here
	})

* The first argument is the command name, as will be shown in the help messages and as will need to be input by the user in the command line to call the command

* The second argument is the command description as will be shown in the help messages

* The third argument is a CmdInitializer, a function that receives a pointer to a Cmd struct representing the command.
In this function, you can add options and arguments by calling the same methods as you would with an app struct (BoolOpt, StringArg, ...).
You would also assign a function to the Action field of the Cmd struct for it to be executed when the command is invoked.

	docker.Command("run", "Run a command in a new container", func(cmd *cli.Cmd) {
		detached := cmd.BoolOpt("d detach", false, "Detached mode: run the container in the background and print the new container ID")
		memory := cmd.StringOpt("m memory", "", "Memory limit (format: <number><optional unit>, where unit = b, k, m or g)")

		image := cmd.StringArg("IMAGE", "", "The image to run")

		cmd.Action = func() {
			if *detached {
				//do something
			}
			runContainer(*image, *detached, *memory)
		}
	})

You can also add sub commands by calling Command on the Cmd struct:

	bzk.Command("job", "actions on jobs", func(cmd *cli.Cmd) {
		cmd.Command("list", "list jobs", listJobs)
		cmd.Command("start", "start a new job", startJob)
		cmd.Command("log", "show a job log", nil)
	})

This could go on to any depth if need be.

As a side-note: it may seem a bit weird the way mow.cli uses a function to initialize a command
instead of just returning the command struct.

The motivation behind this choice is scoping: as with the standard flag package, adding an option or an argument
returns a pointer to a value which will be populated when the app is run.

Since you'll want to store these pointers in variables, and to avoid having dozens of them in the same scope (the main func for example or as global variables),
mow.cli's API was specifically tailored to take a func parameter (called CmdInitializer) which accepts the command struct.

This way, the command specific variables scope is limited to this function.

Spec

An app or command's call syntax can be customized using spec strings.
This can be useful to indicate that an argument is optional for example, or that 2 options are mutually exclusive.

You can set a spec string on:

* The app: to configure the syntax for global options and arguments

* A command: to configure the syntax for that command's options and arguments

In both cases, a spec string is assigned to the Spec field:

	cp := cli.App("cp", "Copy files around")
	cp.Spec = "[-R [-H | -L | -P]]"

And:

	docker := cli.App("docker", "A self-sufficient runtime for linux containers")
	docker.Command("run", "Run a command in a new container", func(cmd *cli.Cmd) {
		cmd.Spec = "[-d|--rm] IMAGE [COMMAND [ARG...]]"
		:
		:
	}

The spec syntax is mostly based on the conventions used in POSIX command line apps help messages and man pages:

Options

You can use both short and long option names in spec strings:
	x.Spec="-f"
And:
	x.Spec="--force"

In both cases, we required that the f or force flag be set

Any option you reference in a spec string MUST be explicitly declared, otherwise mow.cli will panic:

	x.BoolOpt("f force", ...)

Arguments

Arguments are all-uppercased words:
	x.Spec="SRC DST"
This spec string will force the user to pass exactly 2 arguments, SRC and DST

Any argument you reference in a spec string MUST be explicitly declared, otherwise mow.cli will panic:

	x.StringArg("SRC", ...)
	x.StringArg("DST", ...)

Ordering

Except for options, The order of the elements in a spec string is respected and enforced when parsing the command line arguments:

	x.Spec = "-f -g SRC -h DST"

Consecutive options (-f and -g for example) get parsed regardless of the order they are specified in (both "-f=5 -g=6" and "-g=6 -f=5" are valid).

Order between options and arguments is significant (-f and -g must appear before the SRC argument).

Same goes for arguments, where SRC must appear before DST.

Optionality

You can mark iterms as optional in a spec string by enclosing them in squqre brackets :[...]
	x.Spec = "[-x]"

Choice

You can use the | operator to indicate a choice between two or more items
	x.Spec = "--rm | --daemon"
	x.Spec = "-H | -L | -P"
	x.Spec = "-t | DST"

Repetition

You can use the ... postfix operator to mark an element as repeatable:
	x.Spec="SRC..."
	x.Spec="-e..."

Grouping

You can group items using parenthesis. This is useful in combination with the choice and repetition operators (| and ...):
	x.Spec = "(-e COMMAND)... | (-x|-y)"
The parenthesis in the example above serve to mark that it is the sequence of a -e flag followed by an argument that is repeatable, and that
all that is mutually exclusive to a choice betwwen -x and -y options.

Option group

This is a shortcut to declare a choice between multiple options:
	x.Spec = "-abcd"
Is equivalent to:
	x.Spec = "(-a | -b | -c | -d)..."
I.e. any combination of the listed options in any order, with at least one option.

All options

Another shortcut:
	x.Spec = "[OPTIONS]"
This is a special syntax (the square brackets are not for marking an optional item, and the uppercased word is not for an argument).
This is equivalent to a repeatable choice between all the available options.
For example, if an app or a command declares 4 options a, b, c and d, [OPTIONS] is equivalent to
	x.Spec = "[-a | -b | -c | -d]..."


Operators

The `--` operator can be used in a spec string to automatically treat everything following it as an options.

In other words, placing a `--` in the spec string automatically inserts a `--` in the same position in the program call arguments.

This lets you write programs like the `time` utility for example:

	x.Spec = "time -lp [-- CMD [ARG...]]"


Spec Grammar

Here's the EBNF grammar for the Specs language:

	spec         -> sequence
	sequence     -> choice*
	req_sequence -> choice+
	choice       -> atom ('|' atom)*
	atom         -> (shortOpt | longOpt | optSeq | allOpts | group | optional) rep?
	shortOp      -> '-' [A-Za-z]
	longOpt      -> '--' [A-Za-z][A-Za-z0-9]*
	optSeq       -> '-' [A-Za-z]+
	allOpts      -> '[OPTIONS]'
	group        -> '(' req_sequence ')'
	optional     -> '[' req_sequence ']'
	rep          -> '...'

And that's it for the spec language.
You can combine these few building blocks in any way you want (while respecting the grammar above) to construct sophisticated validation constraints
(don't go too wild though).

Behind the scenes, mow.cli parses the spec string and constructs a finite state machine to be used to parse the command line arguments.
mow.cli also handles backtracking, and so it can handle tricky cases, or what I like to call "the cp test"
	cp SRC... DST
Without backtracking, this deceptively simple spec string cannot be parsed correctly.
For instance, docopt can't handle this case, whereas mow.cli does.

Default spec

By default, and unless a spec string is set by the user, mow.cli auto-generates one for the app and every command using this logic:

* Start with an empty spec string

* If at least one option was declared, append "[OPTIONS]" to the spec string

* For every declared argument, append it, in the order of declaration, to the spec string

For example, given this command delcaration:
	docker.Command("run", "Run a command in a new container", func(cmd *cli.Cmd) {
		detached := cmd.BoolOpt("d detach", false, "Detached mode: run the container in the background and print the new container ID")
		memory := cmd.StringOpt("m memory", "", "Memory limit (format: <number><optional unit>, where unit = b, k, m or g)")

		image := cmd.StringArg("IMAGE", "", "")
		args := cmd.StringsArg("ARG", "", "")
	})
The auto-generated spec string would be:
	[OPTIONS] IMAGE ARG

Which should suffice for simple cases. If not, the spec string has to be set explicitly.
*/
package cli
