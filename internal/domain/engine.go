package domain

import (
	"strings"
)

// ParseQuery здійснює швидку екстракцію намірів та сутностей для миттєвої реакції UI.
// При наявності зовнішнього LLM ключа запит делегується до моделі, інакше спрацьовує детермінований grounded matcher.
func (r *Repository) ProcessQuery(domainID, text, lang string) ChatResponse {
	d, ok := r.GetDomain(domainID)
	if !ok {
		d = r.domains["pizzeria"]
		domainID = "pizzeria"
	}

	lower := strings.ToLower(strings.TrimSpace(text))

	resp := ChatResponse{
		Language:  lang,
		DomainID:  domainID,
		UIActions: make([]UIAction, 0),
	}

	// 1. Пошук збігів товарів/послуг у домені
	var matchedItem *Item
	for i := range d.Items {
		item := &d.Items[i]
		titleLower := strings.ToLower(item.Title)
		idLower := strings.ToLower(item.ID)
		if strings.Contains(lower, titleLower) ||
			(strings.Contains(lower, "пепероні") && strings.Contains(idLower, "peperoni")) ||
			(strings.Contains(lower, "маргарит") && strings.Contains(idLower, "margherita")) ||
			(strings.Contains(lower, "чотири сири") && strings.Contains(idLower, "formaggi")) ||
			(strings.Contains(lower, "кол") && strings.Contains(idLower, "cola")) ||
			(strings.Contains(lower, "бород") && strings.Contains(idLower, "beard")) ||
			(strings.Contains(lower, "стрижк") && strings.Contains(idLower, "haircut")) ||
			(strings.Contains(lower, "діагностик") && strings.Contains(idLower, "diag")) ||
			(strings.Contains(lower, "мастил") && strings.Contains(idLower, "oil")) ||
			(strings.Contains(lower, "килимок") && strings.Contains(idLower, "mat")) ||
			(strings.Contains(lower, "валик") && strings.Contains(idLower, "roller")) {
			matchedItem = item
			break
		}
	}

	// 2. Розбір намірів: Додати в кошик / Замовити
	isOrderIntent := strings.Contains(lower, "додай") ||
		strings.Contains(lower, "замов") ||
		strings.Contains(lower, "хочу") ||
		strings.Contains(lower, "візьму") ||
		strings.Contains(lower, "купити") ||
		strings.Contains(lower, "add") ||
		strings.Contains(lower, "order")

	// Намір: Оформити / Оплатити
	isCheckoutIntent := strings.Contains(lower, "оформити") ||
		strings.Contains(lower, "оплат") ||
		strings.Contains(lower, "чек") ||
		strings.Contains(lower, "зірк") ||
		strings.Contains(lower, "stars") ||
		strings.Contains(lower, "checkout")

	// Намір: Ціна / Інфо
	isPriceQuery := strings.Contains(lower, "скільки коштує") ||
		strings.Contains(lower, "ціна") ||
		strings.Contains(lower, "вартість") ||
		strings.Contains(lower, "price") ||
		strings.Contains(lower, "how much")

	if isCheckoutIntent {
		resp.Intent = "open_checkout"
		resp.ReplyText = "Чудово! Відкриваю вікно замовлення для перевірки та швидкої оплати в Telegram Stars ⭐."
		resp.UIActions = append(resp.UIActions, UIAction{
			Type:    "OPEN_CHECKOUT",
			Payload: map[string]interface{}{"domain": domainID},
		})
		return resp
	}

	if matchedItem != nil {
		if isOrderIntent {
			resp.Intent = "add_to_cart"
			resp.ReplyText = "Чудовий вибір! Додав «" + matchedItem.Title + "» до вашого кошика. Бажаєте додати щось ще чи перейти до оплати?"
			resp.UIActions = append(resp.UIActions, UIAction{
				Type: "ADD_TO_CART",
				Payload: map[string]interface{}{
					"item_id":     matchedItem.ID,
					"title":       matchedItem.Title,
					"price_uah":   matchedItem.PriceUAH,
					"price_stars": matchedItem.PriceStars,
					"icon":        matchedItem.Icon,
					"quantity":    1,
				},
			})
			resp.UIActions = append(resp.UIActions, UIAction{
				Type: "HIGHLIGHT_ITEM",
				Payload: map[string]interface{}{
					"item_id":     matchedItem.ID,
					"duration_ms": 2500,
				},
			})
		} else if isPriceQuery {
			resp.Intent = "query_price"
			resp.ReplyText = "Вартість «" + matchedItem.Title + "» становить " + string(rune(matchedItem.PriceUAH)) + " грн або " + string(rune(matchedItem.PriceStars)) + " ⭐ Stars. Додати до замовлення?"
			resp.UIActions = append(resp.UIActions, UIAction{
				Type: "HIGHLIGHT_ITEM",
				Payload: map[string]interface{}{
					"item_id":     matchedItem.ID,
					"duration_ms": 3000,
				},
			})
		} else {
			resp.Intent = "recommendation"
			resp.ReplyText = "Зверніть увагу на «" + matchedItem.Title + "» (" + matchedItem.Description + "). Підсвічую на екрані."
			resp.UIActions = append(resp.UIActions, UIAction{
				Type: "HIGHLIGHT_ITEM",
				Payload: map[string]interface{}{
					"item_id":     matchedItem.ID,
					"duration_ms": 3000,
				},
			})
		}
		return resp
	}

	// 3. Якщо прямих товарних слів немає — перевіряємо загальні сценарії
	if strings.Contains(lower, "гостр") {
		// Рекомендація Пепероні
		resp.Intent = "recommendation"
		resp.ReplyText = "Якщо полюбляєте гостре, однозначно рекомендую нашу фірмову піцу «Пепероні» з пікантною салямі. Підсвітив її в меню!"
		resp.UIActions = append(resp.UIActions, UIAction{
			Type: "HIGHLIGHT_ITEM",
			Payload: map[string]interface{}{
				"item_id":     "pizza_peperoni",
				"duration_ms": 3500,
			},
		})
		return resp
	}

	if strings.Contains(lower, "спин") {
		resp.Intent = "recommendation"
		resp.ReplyText = "Для зняття болю та втоми в спині найкраще підходить килимок «Великий» 7.0 Ag. Він охоплює всю зону хребта. Підсвічую картку!"
		resp.UIActions = append(resp.UIActions, UIAction{
			Type: "HIGHLIGHT_ITEM",
			Payload: map[string]interface{}{
				"item_id":     "lyapko_mat_large",
				"duration_ms": 3500,
			},
		})
		return resp
	}

	// 4. Fallback / Guardrail
	resp.Intent = "general"
	resp.ReplyText = "Я вас почув! Можу відповісти на запитання щодо нашого меню та послуг, підібрати позицію або одразу додати її в кошик. Що оберемо?"
	return resp
}
