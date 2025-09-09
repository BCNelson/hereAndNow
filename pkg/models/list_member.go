package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ListMember struct {
	ID         string     `db:"id" json:"id"`
	ListID     string     `db:"list_id" json:"list_id"`
	UserID     string     `db:"user_id" json:"user_id"`
	Role       MemberRole `db:"role" json:"role"`
	InvitedBy  string     `db:"invited_by" json:"invited_by"`
	InvitedAt  time.Time  `db:"invited_at" json:"invited_at"`
	AcceptedAt *time.Time `db:"accepted_at" json:"accepted_at"`
}

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleEditor MemberRole = "editor"
	MemberRoleViewer MemberRole = "viewer"
)

func NewListMember(listID, userID, invitedBy string, role MemberRole) (*ListMember, error) {
	if listID == "" {
		return nil, fmt.Errorf("list ID is required")
	}

	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	if invitedBy == "" {
		return nil, fmt.Errorf("invited by user ID is required")
	}

	if !isValidMemberRole(role) {
		return nil, fmt.Errorf("invalid member role: %s", role)
	}

	now := time.Now()
	member := &ListMember{
		ID:        uuid.New().String(),
		ListID:    listID,
		UserID:    userID,
		Role:      role,
		InvitedBy: invitedBy,
		InvitedAt: now,
	}

	if role == MemberRoleOwner {
		member.AcceptedAt = &now
	}

	return member, nil
}

func (lm *ListMember) Accept() {
	now := time.Now()
	lm.AcceptedAt = &now
}

func (lm *ListMember) SetRole(role MemberRole) error {
	if !isValidMemberRole(role) {
		return fmt.Errorf("invalid member role: %s", role)
	}
	lm.Role = role
	return nil
}

func (lm *ListMember) IsOwner() bool {
	return lm.Role == MemberRoleOwner
}

func (lm *ListMember) IsEditor() bool {
	return lm.Role == MemberRoleEditor
}

func (lm *ListMember) IsViewer() bool {
	return lm.Role == MemberRoleViewer
}

func (lm *ListMember) CanEdit() bool {
	return lm.Role == MemberRoleOwner || lm.Role == MemberRoleEditor
}

func (lm *ListMember) CanView() bool {
	return lm.Role == MemberRoleOwner || lm.Role == MemberRoleEditor || lm.Role == MemberRoleViewer
}

func (lm *ListMember) CanManageMembers() bool {
	return lm.Role == MemberRoleOwner
}

func (lm *ListMember) CanDelete() bool {
	return lm.Role == MemberRoleOwner
}

func (lm *ListMember) HasAccepted() bool {
	return lm.AcceptedAt != nil
}

func (lm *ListMember) IsPending() bool {
	return lm.AcceptedAt == nil
}

func (lm *ListMember) IsUser(userID string) bool {
	return lm.UserID == userID
}

func (lm *ListMember) BelongsToList(listID string) bool {
	return lm.ListID == listID
}

func (lm *ListMember) WasInvitedBy(userID string) bool {
	return lm.InvitedBy == userID
}

func (lm *ListMember) InvitationAge() time.Duration {
	return time.Since(lm.InvitedAt)
}

func (lm *ListMember) MembershipDuration() *time.Duration {
	if lm.AcceptedAt == nil {
		return nil
	}
	duration := time.Since(*lm.AcceptedAt)
	return &duration
}

func (lm *ListMember) Validate() error {
	if lm.ListID == "" {
		return fmt.Errorf("list ID is required")
	}

	if lm.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if lm.InvitedBy == "" {
		return fmt.Errorf("invited by user ID is required")
	}

	if !isValidMemberRole(lm.Role) {
		return fmt.Errorf("invalid member role: %s", lm.Role)
	}

	return nil
}

func isValidMemberRole(role MemberRole) bool {
	switch role {
	case MemberRoleOwner, MemberRoleEditor, MemberRoleViewer:
		return true
	default:
		return false
	}
}