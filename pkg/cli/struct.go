package cli

type CLI struct {
	ShowHidden bool `help:"Show hidden files;" name:"all" short:"a"`
	Recursive  bool `help:"recursive list subdirectories recursively;" short:"R"`

	Version bool `help:"output version information and exit;" name:"version"`

	DBConn string `help:"Db connection string." name:"db-connection"`
	DBType string `help:"Db type." enum:"sqlite,postgres" default:"sqlite" name:"db-type"`
	DBId   string `help:"id to identify the scan" name:"db-id"`

	Hash bool `help:"will create a hash for each file (checksum) and dir (hash of all contents)" name:"hash" `

	Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path" default:"."`
}

var Arguments = CLI{}
