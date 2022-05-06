package cli

type CLI struct {
	Recursive  bool `help:"Recursively remove files. " short:"r"`
	ShowHidden bool `help:"Show hidden files. " name:"all" short:"a"`
	// output    string `enum:"console,database,file" default:"console" short:"o"`
	// DBConn    string `help:"Db connection string."`
	// Format    string `help:"output format using go-template, cannot be used with -o=databse "`

	Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path" default:"."`
}

var Arguments = CLI{}
