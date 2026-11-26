package api

import (
	"context"
	"fmt"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"io"
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

func TestRemovedMemberCannotMutateOwnMessage(t *testing.T) {
	for _, removal := range []string{"kick", "ban"} {
		for _, op := range []string{"edit", "delete"} {
			t.Run(removal+"/"+op, func(t *testing.T) {
				g, m, r, o := permMocks(permissions.PermViewChannel)
				// Kicks and bans both remove the guild membership row.
				m.GetByGuildAndUserFn = func(context.Context, int64, int64) (*models.Member, error) { return nil, nil }
				writes := 0
				msg := &models.MessageWithAuthor{Message: models.Message{ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID, Content: "earlier"}}
				msgs := &mockMessageRepo{
					GetByIDFn: func(context.Context, int64) (*models.MessageWithAuthor, error) { return msg, nil },
					UpdateFn:  func(context.Context, *models.Message) error { writes++; return nil },
					DeleteFn:  func(context.Context, int64) error { writes++; return nil },
				}
				gw := &mockGateway{}
				h := newMessageHandler(msgs, channelMock(), m, r, g, o, gw)
				c, rec := newTestContext(http.MethodPatch, "/", strings.NewReader(`{"content":"edited"}`))
				c.SetParamNames("id", "message_id")
				c.SetParamValues(fmt.Sprint(testChannelID), fmt.Sprint(testMsgID))
				setAuthUser(c, testUserID)
				var err error
				if op == "edit" {
					err = h.EditMessage(c)
				} else {
					err = h.DeleteMessage(c)
				}
				if err != nil {
					t.Fatal(err)
				}
				if rec.Code != 403 || writes != 0 || len(gw.events) != 0 {
					t.Fatalf("removed member: status=%d writes=%d events=%d", rec.Code, writes, len(gw.events))
				}
			})
		}
	}
}

func TestUploadRequiresViewChannel(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(fmt.Sprint(denied), func(t *testing.T) {
			g, m, r, o := permMocks(permissions.PermViewChannel | permissions.PermAttachFiles)
			o.GetByChannelFn = func(_ context.Context, ch int64) ([]models.ChannelOverride, error) {
				if ch == testChannelID {
					return []models.ChannelOverride{{RoleID: testRoleID, Deny: int64(permissions.PermViewChannel)}}, nil
				}
				return nil, nil
			}
			uploads := 0
			store := &mockStorage{UploadFn: func(context.Context, string, io.Reader, int64, string) error { uploads++; return nil }}
			h := newUploadHandler(&mockAttachmentRepo{}, channelMock(), m, r, g, o, store)
			c, rec := newMultipartContext(t, "test.txt", "text/plain", []byte("test"))
			ch := testChannelID + 1
			if denied {
				ch = testChannelID
			}
			c.SetParamNames("id")
			c.SetParamValues(fmt.Sprint(ch))
			setAuthUser(c, testUserID)
			if err := h.Upload(c); err != nil {
				t.Fatal(err)
			}
			want, calls := 201, 1
			if denied {
				want, calls = 403, 0
			}
			if rec.Code != want || uploads != calls {
				t.Fatalf("status=%d uploads=%d", rec.Code, uploads)
			}
		})
	}
}
