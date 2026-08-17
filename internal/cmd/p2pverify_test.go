package cmd

import "testing"

func TestP2PMembersMatch(t *testing.T) {
	const (
		self  = "ou_self"
		other = "ou_other"
		third = "ou_third"
	)

	tests := []struct {
		name     string
		members  []string
		self     string
		target   string
		want     bool
		wantSolo bool
	}{
		{
			name:    "1:1 chat with the target",
			members: []string{self, other},
			self:    self,
			target:  other,
			want:    true,
		},
		{
			name:    "1:1 chat, members in either order",
			members: []string{other, self},
			self:    self,
			target:  other,
			want:    true,
		},
		{
			// The Leo Yan bug: a DM between us and someone else, asked about as if
			// it were our own note-to-self chat. Sender-based inference said yes.
			name:    "our DM with a third party is not our self-chat",
			members: []string{self, other},
			self:    self,
			target:  self,
			want:    false,
		},
		{
			// Membership alone can't tell the note-to-self chat from a bot DM —
			// Lark omits bots from member lists, so both are exactly [self].
			// The caller has to consult owner_id.
			name:     "solo member list is undecided",
			members:  []string{self},
			self:     self,
			target:   self,
			want:     false,
			wantSolo: true,
		},
		{
			name:     "solo member list is undecided for a bot target too",
			members:  []string{self},
			self:     self,
			target:   "ou_bot",
			want:     false,
			wantSolo: true,
		},
		{
			name:    "target is absent",
			members: []string{self, third},
			self:    self,
			target:  other,
			want:    false,
		},
		{
			name:    "group chat containing the target is rejected",
			members: []string{self, other, third},
			self:    self,
			target:  other,
			want:    false,
		},
		{
			name:    "unknown self fails closed",
			members: []string{self, other},
			self:    "",
			target:  other,
			want:    false,
		},
		{
			name:    "empty target fails closed",
			members: []string{self, other},
			self:    self,
			target:  "",
			want:    false,
		},
		{
			name:    "empty member list fails closed",
			members: nil,
			self:    self,
			target:  other,
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, gotSolo := p2pMembersMatch(tc.members, tc.self, tc.target)
			if got != tc.want || gotSolo != tc.wantSolo {
				t.Errorf("p2pMembersMatch(%v, %q, %q) = (%v, solo=%v), want (%v, solo=%v)",
					tc.members, tc.self, tc.target, got, gotSolo, tc.want, tc.wantSolo)
			}
		})
	}
}
