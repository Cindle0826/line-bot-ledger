package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
	"github.com/cindle0826/line-bot-ledger/internal/store"
)

// replier is the slice of the messaging_api client this handler needs.
// Narrowing it down from *messaging_api.MessagingApiAPI lets tests inject a
// fake instead of hitting the real LINE API.
type replier interface {
	ReplyMessage(*messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
}

// webhookHandler handles LINE Messaging API callbacks.
type webhookHandler struct {
	channelSecret string
	bot           replier
	store         store.Store
}

func newWebhookHandler(channelSecret string, bot replier, s store.Store) *webhookHandler {
	return &webhookHandler{channelSecret: channelSecret, bot: bot, store: s}
}

func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cb, err := webhook.ParseRequest(h.channelSecret, r)
	if err != nil {
		if errors.Is(err, webhook.ErrInvalidSignature) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			slog.Error("webhook: parse request failed", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	for _, event := range cb.Events {
		h.handleEvent(r.Context(), event)
	}
}

func (h *webhookHandler) handleEvent(ctx context.Context, event webhook.EventInterface) {
	msgEvent, ok := event.(webhook.MessageEvent)
	if !ok {
		return
	}
	text, ok := msgEvent.Message.(webhook.TextMessageContent)
	if !ok {
		return
	}

	source, ok := msgEvent.Source.(webhook.UserSource)
	if !ok {
		slog.Warn("webhook: message from non-user source, ignoring", "source_type", fmt.Sprintf("%T", msgEvent.Source))
		return
	}

	reply := h.recordEntry(ctx, source.UserId, text.Text)
	if err := h.replyText(msgEvent.ReplyToken, reply); err != nil {
		slog.Error("webhook: reply failed", "error", err)
	}
}

func (h *webhookHandler) recordEntry(ctx context.Context, lineUserID, text string) string {
	entry, err := ledger.ParseMessage(text)
	if errors.Is(err, ledger.ErrNotAnEntry) {
		return "看不懂這筆記帳，格式：金額 分類 [備註]，例如 -100 午餐"
	}

	if err := h.store.AddEntry(ctx, lineUserID, entry); err != nil {
		slog.Error("webhook: add entry failed", "error", err)
		return "記帳失敗，請稍後再試"
	}

	sign := "支出"
	if entry.Amount > 0 {
		sign = "收入"
	}
	return fmt.Sprintf("已記錄 %s：%.0f（%s）", sign, entry.Amount, entry.Category)
}

func (h *webhookHandler) replyText(replyToken, text string) error {
	_, err := h.bot.ReplyMessage(&messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{Text: text},
		},
	})
	return err
}
