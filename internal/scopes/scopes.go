package scopes

import "strings"

// ScopeGroup defines a group of OAuth scopes required for a set of commands
type ScopeGroup struct {
	Name        string   // e.g., "calendar"
	Description string   // e.g., "Calendar events and scheduling"
	Scopes      []string // OAuth scope strings
	Commands    []string // CLI commands that require this group
}

// BaseScope is always required for token refresh
const BaseScope = "offline_access"

// Groups defines all available scope groups
var Groups = map[string]ScopeGroup{
	"calendar": {
		Name:        "calendar",
		Description: "Calendar events and scheduling",
		Scopes:      []string{"calendar:calendar", "calendar:calendar:readonly"},
		Commands:    []string{"cal"},
	},
	"contacts": {
		Name:        "contacts",
		Description: "Company directory lookup",
		Scopes:      []string{"contact:contact.base:readonly", "contact:department.base:readonly", "contact:department.organize:readonly", "contact:user:search"},
		Commands:    []string{"contact"},
	},
	"documents": {
		Name:        "documents",
		Description: "Lark Docs, Drive, and Wiki access",
		Scopes: []string{
			"docx:document:readonly", "docx:document", "docx:document:create",
			"docs:doc:readonly", "docs:document.content:read", "docs:document.comment:read",
			"drive:drive:readonly", "drive:drive",
			"space:document:retrieve",
			"wiki:space:read", "wiki:space:retrieve",
			"wiki:node:read", "wiki:node:create", "wiki:node:retrieve", "wiki:node:copy",
			"wiki:member:create", "wiki:member:update", "wiki:member:retrieve",
		},
		Commands: []string{"doc", "wiki"},
	},
	"bitable": {
		Name:        "bitable",
		Description: "Lark Bitable (database) access",
		Scopes:      []string{"bitable:app:readonly", "bitable:app"},
		Commands:    []string{"bitable"},
	},
	"messages": {
		Name:        "messages",
		Description: "Chat and messaging",
		Scopes: []string{
			"im:message:readonly", "im:message", "im:message:send_as_bot",
			"im:message.send_as_user", "im:resource",
			"im:message:recall", "im:message:update",
			"im:message.group_msg:get_as_user", "im:message.p2p_msg:get_as_user",
			"im:message.pins:read", "im:message.pins:write_only",
			"im:message.reactions:read", "im:message.reactions:write_only",
			"im:chat:create", "im:chat:read", "im:chat:update",
			"im:chat.members:read", "im:chat.members:write_only",
			"search:message",
		},
		Commands: []string{"msg", "chat"},
	},
	"mail": {
		Name:        "mail",
		Description: "Email via IMAP",
		Scopes:      []string{"mail:user_mailbox.message.address:read", "mail:user_mailbox.message.body:read", "mail:user_mailbox.message.subject:read", "mail:user_mailbox.message:readonly"},
		Commands:    []string{"mail"},
	},
	"minutes": {
		Name:        "minutes",
		Description: "Meeting recordings and transcripts",
		Scopes:      []string{"minutes:minutes:readonly", "minutes:minute:download"},
		Commands:    []string{"minutes"},
	},
	"tasks": {
		Name:        "tasks",
		Description: "Lark Tasks management",
		Scopes: []string{
			"task:task:read", "task:task:write",
			"task:tasklist:read", "task:tasklist:write",
			"task:section:read", "task:section:write",
		},
		Commands: []string{"task"},
	},
	"meetings": {
		Name:        "meetings",
		Description: "Video conferencing records and notes",
		Scopes: []string{
			"vc:meeting.search:read", "vc:meeting.meetingevent:read",
			"vc:note:read", "vc:record:readonly",
		},
		Commands: []string{"meetings"},
	},
	"approval": {
		Name:        "approval",
		Description: "Approval workflows",
		Scopes: []string{
			"approval:approval", "approval:approval:readonly",
			"approval:instance:write", "approval:task:write",
		},
		Commands: []string{"approval"},
	},
	"attendance": {
		Name:        "attendance",
		Description: "Attendance clock-in records",
		Scopes: []string{
			"attendance:task", "attendance:task:readonly",
		},
		Commands: []string{"attendance"},
	},
	"slides": {
		Name:        "slides",
		Description: "Lark Slides (presentations)",
		Scopes: []string{
			"slides:presentation:create", "slides:presentation:read",
			"slides:presentation:write_only", "slides:presentation:update",
			"docs:document.media:upload", "drive:drive",
		},
		Commands: []string{"slides"},
	},
}

// AllGroupNames returns all scope group names in a consistent order
func AllGroupNames() []string {
	return []string{"calendar", "contacts", "documents", "bitable", "messages", "mail", "minutes", "tasks", "meetings", "approval", "slides", "attendance"}
}

// GetScopesForGroups returns the combined scopes for the given group names
func GetScopesForGroups(groupNames []string) []string {
	scopeSet := make(map[string]bool)
	scopeSet[BaseScope] = true // Always include base scope

	for _, name := range groupNames {
		if group, ok := Groups[name]; ok {
			for _, scope := range group.Scopes {
				scopeSet[scope] = true
			}
		}
	}

	scopes := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		scopes = append(scopes, scope)
	}
	return scopes
}

// GetAllScopes returns all scopes for all groups (full permissions)
func GetAllScopes() []string {
	return GetScopesForGroups(AllGroupNames())
}

// GetScopeString returns scopes as a space-separated string for OAuth
func GetScopeString(groupNames []string) string {
	scopes := GetScopesForGroups(groupNames)
	return strings.Join(scopes, " ")
}

// GetAllScopeString returns all scopes as a space-separated string
func GetAllScopeString() string {
	return GetScopeString(AllGroupNames())
}

// GetGroupForCommand returns the scope group required by a command
func GetGroupForCommand(cmd string) (ScopeGroup, bool) {
	for _, group := range Groups {
		for _, c := range group.Commands {
			if c == cmd {
				return group, true
			}
		}
	}
	return ScopeGroup{}, false
}

// ParseGroups parses a comma-separated list of group names and validates them
func ParseGroups(input string) ([]string, []string) {
	if input == "" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	valid := make([]string, 0, len(parts))
	invalid := make([]string, 0)

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := Groups[name]; ok {
			valid = append(valid, name)
		} else {
			invalid = append(invalid, name)
		}
	}

	return valid, invalid
}
