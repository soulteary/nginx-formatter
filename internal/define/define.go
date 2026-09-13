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
	// DEFAULT_HOST is empty, meaning every interface. The published Docker
	// usage (`docker run -p 8080:8080 ... serve`) requires this: a container
	// bound to loopback is unreachable from the host.
	DEFAULT_HOST = ""
)

// DISPLAY_INDENT_CHARS maps an effective indent character to a human-readable
// name for the CLI's confirmation message. Only real characters appear here:
// the escape spellings ("\\s", "\\t") are normalized away before lookup.
var DISPLAY_INDENT_CHARS = map[string]string{
	" ":  "[SPACE]",
	"\t": "[TAB]",
}

const (
	APP_ARGV_INPUT  = "input"
	APP_ARGV_OUTPUT = "output"
	APP_ARGV_INDENT = "indent"
	APP_ARGV_CHAR   = "char"
	// web flags
	APP_ARGV_PORT = "port"
	APP_ARGV_WEB  = "web"
	APP_ARGV_HOST = "host"
)
