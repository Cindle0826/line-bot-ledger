package ledger

import (
	"errors"
	"strconv"
	"strings"
)

// ErrNotAnEntry indicates a chat message doesn't look like a bookkeeping
// entry, so the caller should treat it as something else (e.g. a command).
var ErrNotAnEntry = errors.New("ledger: message is not an entry")

// ParseMessage parses a chat message such as "-100 午餐" or "+5000 薪水 七月獎金"
// into an Entry: <amount> <category> [note...]. An explicit "+" marks income;
// a "-" or no sign at all marks an expense, since spending is the common case
// for a personal ledger bot.
func ParseMessage(text string) (Entry, error) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return Entry{}, ErrNotAnEntry
	}

	amountField := fields[0]
	amount, err := strconv.ParseFloat(amountField, 64)
	if err != nil {
		return Entry{}, ErrNotAnEntry
	}
	if !strings.HasPrefix(amountField, "+") && !strings.HasPrefix(amountField, "-") {
		amount = -amount
	}

	return Entry{
		Amount:   amount,
		Category: fields[1],
		Note:     strings.Join(fields[2:], " "),
	}, nil
}
