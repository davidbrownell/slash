package store

// RowStatus is the status for a row.
type RowStatus string

const (
	// Normal is the status for a normal row.
	Normal RowStatus = "NORMAL"
	// Archived is the status for an archived row.
	Archived RowStatus = "ARCHIVED"
)

func (e RowStatus) String() string {
	switch e {
	case Normal:
		return "NORMAL"
	case Archived:
		return "ARCHIVED"
	}
	return ""
}

// Visibility is the type of a visibility.
type Visibility string

const (
	// VisibilityPublic is the PUBLIC visibility.
	VisibilityPublic Visibility = "PUBLIC"
	// VisibilityWorkspace is the WORKSPACE visibility.
	VisibilityWorkspace Visibility = "WORKSPACE"
	// VisibilityPrivate is the PRIVATE visibility.
	VisibilityPrivate Visibility = "PRIVATE"
)

func (e Visibility) String() string {
	switch e {
	case VisibilityPublic:
		return "PUBLIC"
	case VisibilityWorkspace:
		return "WORKSPACE"
	case VisibilityPrivate:
		return "PRIVATE"
	}
	return "PRIVATE"
}
