package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/models"
)

func TestMessagesRoleVisibilityAndReadReceiptIntegration(t *testing.T) {
	router, database, enforcer := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	role := models.Role{Name: "message-ops"}
	if err := database.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	passwordHash, err := auth.HashPassword("Ops@123")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}
	opsUser := models.User{
		Username:     "message-ops-user",
		PasswordHash: passwordHash,
		IsActive:     true,
	}
	if err := database.Create(&opsUser).Error; err != nil {
		t.Fatalf("create ops user failed: %v", err)
	}
	if err := database.Create(&models.UserRole{UserID: opsUser.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("bind ops role failed: %v", err)
	}
	if _, err := enforcer.AddPolicy(role.Name, "/api/v1/messages", "GET", "*", "*", "*"); err != nil {
		t.Fatalf("add messages list policy failed: %v", err)
	}
	if _, err := enforcer.AddPolicy(role.Name, "/api/v1/messages/:id/read", "POST", "*", "*", "*"); err != nil {
		t.Fatalf("add messages read policy failed: %v", err)
	}

	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/messages", adminToken, map[string]any{
		"channel": "role",
		"target":  role.Name,
		"title":   "变更通知",
		"content": "今晚执行灰度发布",
		"data": map[string]any{
			"source": "integration-test",
		},
	})
	createResp := assertOKResponse(t, createRec)
	var created models.InAppMessage
	if err := json.Unmarshal(createResp.Data, &created); err != nil {
		t.Fatalf("unmarshal created message failed: %v", err)
	}
	if created.TraceID == "" || created.Channel != "role" || created.Target != role.Name {
		t.Fatalf("unexpected created message: %+v", created)
	}

	opsToken := loginAndGetToken(t, router, opsUser.Username, "Ops@123")
	listRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/messages?page=1&pageSize=10", opsToken, nil)
	listResp := assertOKResponse(t, listRec)
	var listData listPayload[messageResponseForTest]
	if err := json.Unmarshal(listResp.Data, &listData); err != nil {
		t.Fatalf("unmarshal message list failed: %v", err)
	}
	if listData.Total != 1 || len(listData.List) != 1 {
		t.Fatalf("expected one unread role message, got total=%d list=%d", listData.Total, len(listData.List))
	}
	if listData.List[0].Read {
		t.Fatalf("expected message unread before mark read")
	}

	readRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/messages/%d/read", created.ID), opsToken, nil)
	_ = assertOKResponse(t, readRec)

	unreadRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/messages?read=false&page=1&pageSize=10", opsToken, nil)
	unreadResp := assertOKResponse(t, unreadRec)
	var unreadData listPayload[messageResponseForTest]
	if err := json.Unmarshal(unreadResp.Data, &unreadData); err != nil {
		t.Fatalf("unmarshal unread message list failed: %v", err)
	}
	if unreadData.Total != 0 {
		t.Fatalf("expected no unread messages after read, got total=%d", unreadData.Total)
	}

	readListRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/messages?read=true&page=1&pageSize=10", opsToken, nil)
	readListResp := assertOKResponse(t, readListRec)
	var readListData listPayload[messageResponseForTest]
	if err := json.Unmarshal(readListResp.Data, &readListData); err != nil {
		t.Fatalf("unmarshal read message list failed: %v", err)
	}
	if readListData.Total != 1 || !readListData.List[0].Read || readListData.List[0].ReadAt == "" {
		t.Fatalf("expected one read message with readAt, got %+v", readListData)
	}
}

type messageResponseForTest struct {
	models.InAppMessage
	ReadAt string `json:"readAt"`
}
