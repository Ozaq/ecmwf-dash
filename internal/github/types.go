package github

import "time"

type Issue struct {
    Repository        string
    Number            int
    Title             string
    URL               string
    Author            string
    AuthorAvatar      string
    AuthorAssociation string // OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, NONE
    IsExternal        bool   // true if not OWNER/MEMBER/COLLABORATOR
    CreatedAt         time.Time
    UpdatedAt         time.Time
    Labels            []Label
}

type Label struct {
    Name  string
    Color string
}
