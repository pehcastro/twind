package fix

const (
	ConPTYVersion = "1.25.260930003"
	ConPTYPackage = "https://api.nuget.org/v3-flatcontainer/microsoft.windows.console.conpty/" + ConPTYVersion + "/microsoft.windows.console.conpty." + ConPTYVersion + ".nupkg"
	ConPTYSHA256  = "02b07b349af66d801159bdf9e440d4a1ce78bb951f37fc8609731665afdae7ee"
	PackageFile   = "conpty.nupkg"
	DLL           = "conpty.dll"
	Host          = "OpenConsole.exe"
	DLLEntry      = "runtimes/win-%s/native/conpty.dll"
	HostEntry     = "build/native/runtimes/%s/OpenConsole.exe"
	Manifest      = "twind-conpty.json"
	Backup        = ".bak"
	Alacritty     = "alacritty.exe"
	Rio           = "rio.exe"
	ModernBuild   = 26100
	Ancestors     = 16
	CacheDir      = "twind"
	FilePerm      = 0o644
	DirPerm       = 0o755
)
