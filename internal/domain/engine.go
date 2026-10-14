package domain

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ProcessQuery is the deterministic stub of the engine's dialogue turn: it
// matches catalog keywords and a few intents, and emits UI actions. The real
// engine (v2v-demo) replaces it behind the same /v1 contract; nothing here
// calls an LLM.
func (r *Repository) ProcessQuery(domainID, text, lang string) ChatResponse {
	d, ok := r.GetDomain(domainID)
	if !ok {
		d, domainID = r.domains["pizzeria"], "pizzeria"
	}
	resp := ChatResponse{Language: lang, Signal: "continue", UIActions: []UIAction{}}

	norm := normalize(text)

	if hasAny(norm, "оформити", "оплат", "чек", "зірк", "stars", "checkout") {
		resp.Intent = "open_checkout"
		resp.Reply = "Чудово! Відкриваю вікно замовлення для перевірки та швидкої оплати в Telegram Stars ⭐."
		resp.UIActions = append(resp.UIActions, UIAction{Type: "OPEN_CHECKOUT", Payload: ActionPayload{Domain: domainID}})
		return resp
	}

	matches := matchItems(d.Items, norm)
	if len(matches) == 0 {
		resp.Intent = "general"
		resp.Reply = "Я вас почув! Можу відповісти на запитання щодо нашого меню та послуг, підібрати позицію або одразу додати її в кошик. Що оберемо?"
		return resp
	}

	isOrder := hasAny(norm, "додай", "запиш", "замов", "хочу", "візьму", "купити", "add", "order")
	isPrice := hasAny(norm, "скільки коштує", "ціна", "вартість", "price", "how much")

	switch {
	case isOrder:
		resp.Intent = "add_to_cart"
		titles := make([]string, 0, len(matches))
		for _, m := range matches {
			titles = append(titles, "«"+m.item.Title+"»")
			resp.UIActions = append(resp.UIActions, UIAction{Type: "ADD_TO_CART", Payload: ActionPayload{ItemID: m.item.ID, Quantity: m.qty}})
		}
		last := matches[len(matches)-1].item
		resp.UIActions = append(resp.UIActions, UIAction{Type: "HIGHLIGHT_ITEM", Payload: ActionPayload{ItemID: last.ID, DurationMS: 2500}})
		resp.Reply = "Чудовий вибір! Додав " + strings.Join(titles, " та ") + " до вашого кошика. Бажаєте додати щось ще чи перейти до оплати?"
	case isPrice:
		resp.Intent = "query_price"
		parts := make([]string, 0, len(matches))
		for _, m := range matches {
			parts = append(parts, "«"+m.item.Title+"» — "+strconv.Itoa(m.item.PriceUAH)+" грн або "+strconv.Itoa(m.item.PriceStars)+" ⭐ Stars")
		}
		resp.Reply = "Вартість: " + strings.Join(parts, "; ") + ". Додати до замовлення?"
		resp.UIActions = append(resp.UIActions, UIAction{Type: "HIGHLIGHT_ITEM", Payload: ActionPayload{ItemID: matches[0].item.ID, DurationMS: 3000}})
	default:
		resp.Intent = "recommendation"
		m := matches[0].item
		resp.Reply = "Зверніть увагу на «" + m.Title + "» (" + m.Description + "). Підсвічую на екрані."
		resp.UIActions = append(resp.UIActions, UIAction{Type: "HIGHLIGHT_ITEM", Payload: ActionPayload{ItemID: m.ID, DurationMS: 3000}})
	}
	return resp
}

type itemMatch struct {
	item Item
	qty  int
}

type span struct {
	item       Item
	start, end int
}

// matchItems finds catalog items whose keywords (word prefixes, or phrases)
// occur in norm. When two keywords overlap, the longer one wins ("стрижку
// бороди" beats "стрижк"), so one phrase never matches two items. The quantity
// is a digit or numeral word right before the keyword, default 1.
func matchItems(items []Item, norm string) []itemMatch {
	padded := []rune(" " + norm + " ")
	var spans []span
	for _, it := range items {
		kws := append([]string{strings.ToLower(it.Title)}, it.Keywords...)
		for _, kw := range kws {
			needle := []rune(" " + strings.ToLower(kw))
			if len(needle) < 3 {
				continue
			}
			for i := 0; i+len(needle) <= len(padded); i++ {
				if runesEqual(padded[i:i+len(needle)], needle) {
					spans = append(spans, span{item: it, start: i + 1, end: i + len(needle)})
				}
			}
		}
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].end-spans[i].start > spans[j].end-spans[j].start })

	var accepted []span
	for _, s := range spans {
		overlap := false
		for _, a := range accepted {
			if s.start < a.end && a.start < s.end {
				overlap = true
				break
			}
		}
		if !overlap {
			accepted = append(accepted, s)
		}
	}
	sort.Slice(accepted, func(i, j int) bool { return accepted[i].start < accepted[j].start })

	var out []itemMatch
	seen := map[string]int{}
	for _, s := range accepted {
		q := quantityBefore(padded, s.start)
		if idx, ok := seen[s.item.ID]; ok {
			out[idx].qty = min(out[idx].qty+q, MaxLineQty)
			continue
		}
		seen[s.item.ID] = len(out)
		out = append(out, itemMatch{item: s.item, qty: q})
	}
	return out
}

var numerals = map[string]int{
	"один": 1, "одну": 1, "одна": 1, "одне": 1, "one": 1,
	"два": 2, "дві": 2, "two": 2,
	"три": 3, "three": 3,
	"чотири": 4, "four": 4,
	"п'ять": 5, "five": 5,
}

// quantityBefore reads the token directly before the rune offset at.
func quantityBefore(padded []rune, at int) int {
	end := at - 1 // padded[end] is the space before the keyword
	if end <= 0 {
		return 1
	}
	start := end
	for start > 0 && padded[start-1] != ' ' {
		start--
	}
	tok := string(padded[start:end])
	if n, err := strconv.Atoi(tok); err == nil && n >= 1 {
		return min(n, MaxLineQty)
	}
	if n, ok := numerals[tok]; ok {
		return n
	}
	return 1
}

func runesEqual(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// normalize lower-cases and replaces punctuation with spaces, unifying the two
// apostrophe forms so "п'ять" and "пʼять" compare equal.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == 'ʼ' || r == '’':
			b.WriteRune('\'')
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'':
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func hasAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
