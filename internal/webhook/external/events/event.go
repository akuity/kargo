package events

// Event contains the genericized read-only event information.
type Event interface {
	// Repository designates which repository the event is for
	// It is the full name of the repository, e.g. https://github.com/username/repo
	Repository() string
	// PushedBy designates is the username of the user who pushed the commit
	PushedBy() string
	// Commit returns the head commit hash
	Commit() string
}
