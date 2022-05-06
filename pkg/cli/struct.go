package cli

type CLI struct {
	Recursive  bool   `help:"Recursively remove files. " short:"r"`
	ShowHidden bool   `help:"Show hidden files. " name:"all" short:"a"`
	Color      string `enum:"always,auto,never" help:"colorize the output; WHEN can be 'always' (default if omitted), 'auto', or 'never';" name:"color" default:"always" `

	List   bool   `help:"output format using go-template, cannot be used with -o=databse " name:"list" short:"l"`
	Format string `help:"output format using go-template, cannot be used with -o=databse "`
	// output     string `enum:"console,database,file" default:"console" short:"o"`
	// DBConn    string `help:"Db connection string."`

	Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path" default:"."`
}

var Arguments = CLI{}
