package factory

import "github.com/spf13/cobra"

// annotationAuth is the cobra.Command.Annotations key under which a command
// declares which kind of API client it needs. The value is one of the
// AuthMode constants below; an absent or empty value means "no client".
const annotationAuth = "100x.auth"

// AuthMode is what kind of API client a command requires before its RunE
// fires.
type AuthMode string

const (
	// AuthNone means the command does not need an API client at all. Used by
	// help, version, completion, and profile-management commands.
	AuthNone AuthMode = ""

	// AuthPublic means the command needs an unsigned client wired to the
	// public endpoint. Used by the market subtree.
	AuthPublic AuthMode = "public"

	// AuthPrivate means the command needs a signed client built from the
	// active profile's credentials.
	AuthPrivate AuthMode = "private"
)

// RequirePublic marks cmd as needing an unsigned public client. Child
// commands inherit this declaration unless they set their own.
func RequirePublic(cmd *cobra.Command) { setAuth(cmd, AuthPublic) }

// RequirePrivate marks cmd as needing a signed private client. Child
// commands inherit this declaration unless they set their own.
func RequirePrivate(cmd *cobra.Command) { setAuth(cmd, AuthPrivate) }

func setAuth(cmd *cobra.Command, m AuthMode) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[annotationAuth] = string(m)
}

// LookupAuth returns the AuthMode declared by cmd or its nearest ancestor.
// A command without an explicit declaration inherits its parent's; a tree
// with no declarations at all returns AuthNone.
func LookupAuth(cmd *cobra.Command) AuthMode {
	for cur := cmd; cur != nil; cur = cur.Parent() {
		if v, ok := cur.Annotations[annotationAuth]; ok {
			return AuthMode(v)
		}
	}
	return AuthNone
}
