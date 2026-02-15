package permissions

import (
	"strings"
	"testing"
)

func TestHas(t *testing.T) {
	p := PermViewChannel | PermSendMessages
	if !p.Has(PermViewChannel) {
		t.Error("expected Has(PermViewChannel) to be true")
	}
	if !p.Has(PermSendMessages) {
		t.Error("expected Has(PermSendMessages) to be true")
	}
	if p.Has(PermManageMessages) {
		t.Error("expected Has(PermManageMessages) to be false")
	}
}

func TestHasMultiple(t *testing.T) {
	p := PermViewChannel | PermSendMessages | PermManageMessages
	if !p.Has(PermViewChannel | PermSendMessages) {
		t.Error("expected Has(ViewChannel|SendMessages) to be true")
	}
	if p.Has(PermViewChannel | PermManageRoles) {
		t.Error("expected Has(ViewChannel|ManageRoles) to be false when ManageRoles is missing")
	}
}

func TestAdd(t *testing.T) {
	p := PermViewChannel
	p = p.Add(PermSendMessages)
	if !p.Has(PermSendMessages) {
		t.Error("expected permission to be added")
	}
	if !p.Has(PermViewChannel) {
		t.Error("expected original permission to remain")
	}
}

func TestRemove(t *testing.T) {
	p := PermViewChannel | PermSendMessages
	p = p.Remove(PermSendMessages)
	if p.Has(PermSendMessages) {
		t.Error("expected permission to be removed")
	}
	if !p.Has(PermViewChannel) {
		t.Error("expected other permission to remain")
	}
}

func TestRemoveDoesNotAffectOtherBits(t *testing.T) {
	p := PermAllText
	p = p.Remove(PermManageMessages)
	if p.Has(PermManageMessages) {
		t.Error("expected ManageMessages to be removed")
	}
	if !p.Has(PermViewChannel) {
		t.Error("expected ViewChannel to remain")
	}
	if !p.Has(PermSendMessages) {
		t.Error("expected SendMessages to remain")
	}
}

func TestAdministratorBypass(t *testing.T) {
	p := PermAdministrator
	// Administrator should be detectable via Has
	if !p.Has(PermAdministrator) {
		t.Error("expected Administrator bit to be set")
	}
	// But Administrator alone does NOT have other bits set
	if p.Has(PermSendMessages) {
		t.Error("Administrator bit alone should not imply SendMessages; resolver handles bypass")
	}
}

func TestDefaultEveryonePerms(t *testing.T) {
	expected := PermViewChannel | PermSendMessages | PermReadMessageHistory |
		PermConnect | PermSpeak | PermCreateInvite | PermChangeNickname
	if DefaultEveryonePerms != expected {
		t.Errorf("DefaultEveryonePerms = %d, want %d", DefaultEveryonePerms, expected)
	}
}

func TestPermAllContainsAll(t *testing.T) {
	if !PermAll.Has(PermAdministrator) {
		t.Error("PermAll should include Administrator")
	}
	if !PermAll.Has(PermAllText) {
		t.Error("PermAll should include AllText")
	}
	if !PermAll.Has(PermAllVoice) {
		t.Error("PermAll should include AllVoice")
	}
}

func TestConvenienceSets(t *testing.T) {
	if !PermAllText.Has(PermViewChannel) {
		t.Error("AllText should include ViewChannel")
	}
	if !PermAllText.Has(PermSendMessages) {
		t.Error("AllText should include SendMessages")
	}
	if !PermAllText.Has(PermManageMessages) {
		t.Error("AllText should include ManageMessages")
	}
	if PermAllText.Has(PermConnect) {
		t.Error("AllText should not include voice permission Connect")
	}

	if !PermAllVoice.Has(PermConnect) {
		t.Error("AllVoice should include Connect")
	}
	if !PermAllVoice.Has(PermSpeak) {
		t.Error("AllVoice should include Speak")
	}
	if PermAllVoice.Has(PermSendMessages) {
		t.Error("AllVoice should not include text permission SendMessages")
	}
}

func TestAddIdempotent(t *testing.T) {
	p := PermViewChannel
	p = p.Add(PermViewChannel)
	if p != PermViewChannel {
		t.Error("adding the same permission twice should be idempotent")
	}
}

func TestRemoveAbsent(t *testing.T) {
	p := PermViewChannel
	p = p.Remove(PermManageGuild)
	if !p.Has(PermViewChannel) {
		t.Error("removing absent permission should not affect existing ones")
	}
	if p.Has(PermManageGuild) {
		t.Error("ManageGuild should still not be present")
	}
}

func TestString_None(t *testing.T) {
	p := Permission(0)
	if p.String() != "NONE" {
		t.Errorf("expected NONE, got %s", p.String())
	}
}

func TestString_Single(t *testing.T) {
	s := PermViewChannel.String()
	if s != "VIEW_CHANNEL" {
		t.Errorf("expected VIEW_CHANNEL, got %s", s)
	}
}

func TestString_Multiple(t *testing.T) {
	p := PermViewChannel | PermSendMessages
	s := p.String()
	if !strings.Contains(s, "VIEW_CHANNEL") {
		t.Error("expected String to contain VIEW_CHANNEL")
	}
	if !strings.Contains(s, "SEND_MESSAGES") {
		t.Error("expected String to contain SEND_MESSAGES")
	}
}

func TestString_Administrator(t *testing.T) {
	s := PermAdministrator.String()
	if s != "ADMINISTRATOR" {
		t.Errorf("expected ADMINISTRATOR, got %s", s)
	}
}
