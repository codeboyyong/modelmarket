package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSummarizeConversationLabelsRolesAndCapsRunes(t *testing.T) {
	messages := []storedContextMessage{
		{role: "user", content: "  first   question  "},
		{role: "assistant", content: "first answer"},
	}
	summary := summarizeConversation(messages, 32)
	if !strings.HasPrefix(summary, "USER: first question") {
		t.Fatalf("unexpected summary: %q", summary)
	}
	if len([]rune(summary)) > 32 {
		t.Fatalf("summary exceeded rune cap: %d", len([]rune(summary)))
	}
}

func TestWithConversationContextIncludesSummaryPinnedRecentAndIncoming(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select exists").WithArgs("conversation-1", "project-1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("select summary from user_conversation_summaries").WithArgs("conversation-1", "branch-1").WillReturnRows(sqlmock.NewRows([]string{"summary"}).AddRow("Earlier decisions."))
	mock.ExpectQuery("select id, role, content, pinned from user_messages").WithArgs("conversation-1", "branch-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "content", "pinned"}).AddRow("pinned-1", "user", "Always use metric units.", true))
	mock.ExpectQuery("select id, role, content, pinned from user_messages").WithArgs("conversation-1", "branch-1", maxRecentContextMessages).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "content", "pinned"}).
			AddRow("recent-2", "assistant", "Most recent answer.", false).
			AddRow("recent-1", "user", "Recent question.", false))

	result, count, err := app.withConversationContext(context.Background(), "project-1", "conversation-1", "branch-1", []chatMessage{{Role: "system", Content: "File context."}, {Role: "user", Content: "New question."}}, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 || len(result) != 6 {
		t.Fatalf("count=%d result=%#v", count, result)
	}
	got := []string{result[0].Text(), result[1].Text(), result[2].Text(), result[3].Text(), result[4].Text(), result[5].Text()}
	want := []string{"Conversation summary", "File context", "Always use metric", "Recent question", "Most recent answer", "New question"}
	for index := range want {
		if !strings.Contains(got[index], want[index]) {
			t.Fatalf("position %d = %q, want substring %q", index, got[index], want[index])
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithConversationContextPrioritizesNewestRecentMessages(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select exists").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("select id from user_conversation_branches").WithArgs("conversation-1").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("main-branch"))
	mock.ExpectQuery("select summary from user_conversation_summaries").WithArgs("conversation-1", "main-branch").WillReturnRows(sqlmock.NewRows([]string{"summary"}))
	mock.ExpectQuery("select id, role, content, pinned from user_messages").WithArgs("conversation-1", "main-branch").
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "content", "pinned"}))
	mock.ExpectQuery("select id, role, content, pinned from user_messages").WithArgs("conversation-1", "main-branch", maxRecentContextMessages).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "content", "pinned"}).
			AddRow("new", "assistant", "newest", false).
			AddRow("old", "user", "oldest", false))
	result, count, err := app.withConversationContext(context.Background(), "project-1", "conversation-1", "", []chatMessage{{Role: "user", Content: "now"}}, 11)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(result) != 2 || result[0].Text() != "newest" || result[1].Text() != "now" {
		t.Fatalf("count=%d result=%#v", count, result)
	}
}

func TestUpdateMessagePinned(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("update user_messages set pinned").WithArgs("message-1", true).WillReturnResult(sqlmock.NewResult(0, 1))
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/message-1", bytes.NewBufferString(`{"pinned":true}`))
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"pinned":true`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshConversationSummaryUpsertsBranchSummary(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	rows := sqlmock.NewRows([]string{"id", "role", "content", "pinned"}).
		AddRow("message-1", "user", "Question", false).
		AddRow("message-2", "assistant", "Answer", false)
	for index := 0; index < maxRecentContextMessages; index++ {
		rows.AddRow("recent-"+string(rune('a'+index)), "user", "Recent", false)
	}
	mock.ExpectQuery("select id, role, content, pinned from user_messages").WithArgs("conversation-1", "branch-1").WillReturnRows(rows)
	mock.ExpectExec("insert into user_conversation_summaries").
		WithArgs(sqlmock.AnyArg(), "conversation-1", "branch-1", "message-2", "USER: Question\nASSISTANT: Answer", 2, 8).
		WillReturnResult(sqlmock.NewResult(0, 1))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-summaries/refresh", bytes.NewBufferString(`{"conversation_id":"conversation-1","branch_id":"branch-1"}`))
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"message_count":2`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMessagesReturnsPinnedState(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select id, role, content").WithArgs("conversation-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "role", "content", "model_profile_id", "branch_id", "inference_request_id", "customer_charge", "provider_cost", "metadata", "pinned", "created_at"}).
			AddRow("message-1", "user", "Keep this", "", "", "", 0, 0, "{}", true, time.Now()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?conversation_id=conversation-1", nil)
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"pinned":true`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
