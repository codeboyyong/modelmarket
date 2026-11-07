package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	defaultConversationContextTokens = 10000
	maxConversationSummaryRunes      = 6000
	maxRecentContextMessages         = 40
)

type storedContextMessage struct {
	id      string
	role    string
	content string
	pinned  bool
}

func branchKey(branchID string) string { return strings.TrimSpace(branchID) }

func (a *App) updateMessagePinned(w http.ResponseWriter, r *http.Request) {
	messageID := strings.TrimSpace(r.PathValue("id"))
	var req struct {
		Pinned *bool `json:"pinned"`
	}
	if messageID == "" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Pinned == nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_message_update"})
		return
	}
	result, err := a.DB.ExecContext(r.Context(), `update user_messages set pinned = $2 where id = $1`, messageID, *req.Pinned)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "message_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, response{"message": response{"id": messageID, "pinned": *req.Pinned}})
}

func (a *App) conversationSummary(w http.ResponseWriter, r *http.Request) {
	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
	key := branchKey(r.URL.Query().Get("branch_id"))
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_conversation_id"})
		return
	}
	var summary, throughMessageID string
	var messageCount, tokenEstimate int
	err := a.DB.QueryRowContext(r.Context(), `
		select summary, coalesce(through_message_id, ''), message_count, token_estimate
		from user_conversation_summaries where conversation_id = $1 and branch_key = $2`, conversationID, key).
		Scan(&summary, &throughMessageID, &messageCount, &tokenEstimate)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, response{"error": "summary_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"summary": response{"conversation_id": conversationID, "branch_id": key, "through_message_id": throughMessageID, "summary": summary, "message_count": messageCount, "token_estimate": tokenEstimate}})
}

func (a *App) refreshConversationSummary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID string `json:"conversation_id"`
		BranchID       string `json:"branch_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.ConversationID) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_summary_request"})
		return
	}
	messages, err := a.loadStoredContextMessages(r.Context(), req.ConversationID, req.BranchID, false, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if len(messages) == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "conversation_has_no_messages"})
		return
	}
	summaryMessages := messages
	if len(summaryMessages) > maxRecentContextMessages {
		summaryMessages = summaryMessages[:len(summaryMessages)-maxRecentContextMessages]
	} else {
		summaryMessages = nil
	}
	summary := summarizeConversation(summaryMessages, maxConversationSummaryRunes)
	throughID := ""
	var throughValue any
	if len(summaryMessages) > 0 {
		throughID = summaryMessages[len(summaryMessages)-1].id
		throughValue = throughID
	}
	key := branchKey(req.BranchID)
	id := "summary_" + randomHex(12)
	_, err = a.DB.ExecContext(r.Context(), `
		insert into user_conversation_summaries(id, conversation_id, branch_id, branch_key, through_message_id, summary, message_count, token_estimate)
		values($1, $2, nullif($3, ''), $3, $4, $5, $6, $7)
		on conflict(conversation_id, branch_key) do update set
			branch_id = excluded.branch_id, through_message_id = excluded.through_message_id,
			summary = excluded.summary, message_count = excluded.message_count,
			token_estimate = excluded.token_estimate, updated_at = current_timestamp`,
		id, req.ConversationID, key, throughValue, summary, len(summaryMessages), approximateTokens(summary))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"summary": response{"conversation_id": req.ConversationID, "branch_id": key, "through_message_id": throughID, "summary": summary, "message_count": len(summaryMessages), "token_estimate": approximateTokens(summary)}})
}

func (a *App) withConversationContext(ctx context.Context, projectID, conversationID, branchID string, incoming []chatMessage, tokenBudget int) ([]chatMessage, int, error) {
	if conversationID == "" {
		return incoming, 0, nil
	}
	if tokenBudget <= 0 {
		tokenBudget = defaultConversationContextTokens
	}
	var belongs bool
	err := a.DB.QueryRowContext(ctx, `select exists(select 1 from user_conversations where id = $1 and project_id = $2)`, conversationID, projectID).Scan(&belongs)
	if err != nil || !belongs {
		if err == nil {
			err = sql.ErrNoRows
		}
		return nil, 0, err
	}
	if branchKey(branchID) == "" {
		err = a.DB.QueryRowContext(ctx, `
			select id from user_conversation_branches
			where conversation_id = $1
			order by created_at asc, id asc limit 1`, conversationID).Scan(&branchID)
		if err != nil {
			return nil, 0, err
		}
	}
	key := branchKey(branchID)
	var summary string
	err = a.DB.QueryRowContext(ctx, `select summary from user_conversation_summaries where conversation_id = $1 and branch_key = $2`, conversationID, key).Scan(&summary)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, 0, err
	}
	pinned, err := a.loadStoredContextMessages(ctx, conversationID, branchID, true, 0)
	if err != nil {
		return nil, 0, err
	}
	recent, err := a.loadStoredContextMessages(ctx, conversationID, branchID, false, maxRecentContextMessages)
	if err != nil {
		return nil, 0, err
	}

	remaining := tokenBudget
	for _, message := range incoming {
		remaining -= approximateTokens(message.Text()) + 4
	}
	result := make([]chatMessage, 0, len(incoming)+len(pinned)+len(recent)+1)
	count := 0
	selectedPinned := make([]chatMessage, 0, len(pinned))
	for _, message := range pinned {
		cost := approximateTokens(message.content) + 4
		if cost > remaining {
			continue
		}
		selectedPinned = append(selectedPinned, chatMessage{Role: normalizedContextRole(message.role), Content: message.content})
		remaining -= cost
		count++
	}
	var summaryMessage *chatMessage
	if strings.TrimSpace(summary) != "" {
		text := "Conversation summary (use as prior context; newer messages take precedence):\n" + summary
		if cost := approximateTokens(text) + 4; cost <= remaining {
			message := chatMessage{Role: "system", Content: text}
			summaryMessage = &message
			remaining -= cost
			count++
		}
	}
	if summaryMessage != nil {
		result = append(result, *summaryMessage)
	}
	leadingSystemCount := 0
	for leadingSystemCount < len(incoming) && incoming[leadingSystemCount].Role == "system" {
		result = append(result, incoming[leadingSystemCount])
		leadingSystemCount++
	}
	result = append(result, selectedPinned...)
	selectedRecent := make([]chatMessage, 0, len(recent))
	for index := len(recent) - 1; index >= 0; index-- {
		message := recent[index]
		cost := approximateTokens(message.content) + 4
		if cost > remaining {
			continue
		}
		selectedRecent = append(selectedRecent, chatMessage{Role: normalizedContextRole(message.role), Content: message.content})
		remaining -= cost
		count++
	}
	for left, right := 0, len(selectedRecent)-1; left < right; left, right = left+1, right-1 {
		selectedRecent[left], selectedRecent[right] = selectedRecent[right], selectedRecent[left]
	}
	result = append(result, selectedRecent...)
	result = append(result, incoming[leadingSystemCount:]...)
	return result, count, nil
}

func (a *App) loadStoredContextMessages(ctx context.Context, conversationID, branchID string, pinnedOnly bool, limit int) ([]storedContextMessage, error) {
	args := []any{conversationID}
	branchClause := ""
	if key := branchKey(branchID); key != "" {
		branchClause = " and branch_id = $2"
		args = append(args, key)
	}
	pinnedClause := ""
	if pinnedOnly {
		pinnedClause = " and pinned = true"
	} else if limit > 0 {
		pinnedClause = " and pinned = false"
	}
	order := " order by created_at asc, case role when 'user' then 0 when 'assistant' then 1 else 2 end, id asc"
	if limit > 0 {
		args = append(args, limit)
		order = fmt.Sprintf(" order by created_at desc, case role when 'assistant' then 0 when 'user' then 1 else 2 end, id desc limit $%d", len(args))
	}
	rows, err := a.DB.QueryContext(ctx, `select id, role, content, pinned from user_messages where conversation_id = $1`+branchClause+pinnedClause+order, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []storedContextMessage{}
	for rows.Next() {
		var message storedContextMessage
		if err := rows.Scan(&message.id, &message.role, &message.content, &message.pinned); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if limit > 0 {
		for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
			messages[left], messages[right] = messages[right], messages[left]
		}
	}
	return messages, rows.Err()
}

func summarizeConversation(messages []storedContextMessage, maxRunes int) string {
	var builder strings.Builder
	for _, message := range messages {
		content := strings.Join(strings.Fields(message.content), " ")
		if content == "" {
			continue
		}
		line := strings.ToUpper(normalizedContextRole(message.role)) + ": " + content + "\n"
		remaining := maxRunes - utf8.RuneCountInString(builder.String())
		if remaining <= 0 {
			break
		}
		runes := []rune(line)
		if len(runes) > remaining {
			builder.WriteString(string(runes[:remaining]))
			break
		}
		builder.WriteString(line)
	}
	return strings.TrimSpace(builder.String())
}

func approximateTokens(text string) int {
	runes := utf8.RuneCountInString(text)
	if runes == 0 {
		return 0
	}
	return (runes + 3) / 4
}

func normalizedContextRole(role string) string {
	switch role {
	case "assistant", "system", "tool":
		return role
	default:
		return "user"
	}
}
