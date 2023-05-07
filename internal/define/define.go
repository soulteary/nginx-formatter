package define

// default config
const (
	// common config
	DEFAULT_INDENT_SIZE = 2
	DEFAULT_INDENT_CHAR = " "
	DEFAULT_WORKDIR     = ""
	// web config
	DEFAULT_PORT = 8080
	DEFAULT_WEB  = false
)

var DISPLAY_INDENT_CHARS = map[string]string{
	" ":   "[SPACE]",
	"\\s": "[SPACE]",
	"\t":  "[TAB]",
}

const (
	APP_ARGV_INPUT  = "input"
	APP_ARGV_OUTPUT = "output"
	APP_ARGV_INDENT = "indent"
	APP_ARGV_CHAR   = "char"
	// web flags
	APP_ARGV_PORT = "port"
	APP_ARGV_WEB  = "web"
)
