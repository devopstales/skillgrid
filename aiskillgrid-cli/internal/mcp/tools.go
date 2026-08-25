package mcp

// Tool defines an MCP tool with its configuration
type Tool struct {
	Name        string
	Description string
}

var Tools = map[string]*Tool{
	"context7-http": {
		Name:        "Context7 HTTP",
		Description: "Fetch up-to-date library documentation and code examples (MCP)",
	},
	"deepwiki-http": {
		Name:        "DeepWiki HTTP",
		Description: "AI-powered GitHub repository documentation (MCP)",
	},
	"exa-http": {
		Name:        "Exa HTTP",
		Description: "Web search and content retrieval (MCP)",
	},
	"engram": {
		Name:        "Engram",
		Description: "Persistent memory across sessions (MCP)",
	},
	"ccc": {
		Name:        "CCC (Coconaut Code Index)",
		Description: "Semantic code indexing (MCP)",
	},
	"gitnexus": {
		Name:        "Gitnexus",
		Description: "Git repository analysis tool (MCP)",
	},
	"trivy": {
		Name:        "Trivy",
		Description: "Security vulnerability scanner (MCP)",
	},
}
