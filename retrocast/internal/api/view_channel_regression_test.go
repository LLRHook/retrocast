package api

import (
	"context"
	"fmt"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"net/http"
	"strings"
	"testing"
)

func TestMessageViewChannel(t *testing.T) {
	for _, role := range []bool{false, true} {
		for _, op := range []string{"list", "get", "send", "edit", "delete", "typing"} {
			for _, denied := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/role=%v/denied=%v", op, role, denied), func(t *testing.T) {
					g, m, r, o := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory | permissions.PermSendMessages)
					overrideRole := testRoleID
					if role {
						overrideRole++
						r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
							return []models.Role{{ID: overrideRole}}, nil
						}
					}
					o.GetByChannelFn = func(_ context.Context, ch int64) ([]models.ChannelOverride, error) {
						if ch == testChannelID {
							return []models.ChannelOverride{{RoleID: overrideRole, Deny: int64(permissions.PermViewChannel)}}, nil
						}
						return nil, nil
					}
					ch := testChannelID + 1
					if denied {
						ch = testChannelID
					}
					msg := &models.MessageWithAuthor{Message: models.Message{ID: testMsgID, ChannelID: ch, AuthorID: testUserID, Content: "hello"}}
					msgs := &mockMessageRepo{GetByIDFn: func(context.Context, int64) (*models.MessageWithAuthor, error) { return msg, nil }}
					h := newMessageHandler(msgs, channelMock(), m, r, g, o, &mockGateway{})
					c, rec := newTestContext(http.MethodPost, "/", strings.NewReader(`{"content":"hello"}`))
					c.SetParamNames("id", "message_id")
					c.SetParamValues(fmt.Sprint(ch), fmt.Sprint(testMsgID))
					setAuthUser(c, testUserID)
					var err error
					switch op {
					case "list":
						err = h.GetMessages(c)
					case "get":
						err = h.GetMessage(c)
					case "send":
						err = h.SendMessage(c)
					case "edit":
						err = h.EditMessage(c)
					case "delete":
						err = h.DeleteMessage(c)
					case "typing":
						err = h.Typing(c)
					}
					if err != nil {
						t.Fatal(err)
					}
					if denied {
						if rec.Code != 403 || !strings.Contains(rec.Body.String(), "MISSING_PERMISSIONS") {
							t.Fatalf("denied: %d %s", rec.Code, rec.Body.String())
						}
					} else if rec.Code >= 400 {
						t.Fatalf("allowed: %d %s", rec.Code, rec.Body.String())
					}
				})
			}
		}
	}
}
